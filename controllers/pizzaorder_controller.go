/*

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	kErr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/rudoi/pizza-go/pkg/pizza"

	alphav1 "github.com/rudoi/cruster-api/api/v1"
)

// PizzaOrderReconciler reconciles a PizzaOrder object
type PizzaOrderReconciler struct {
	client.Client
	PizzaClient *pizza.Client
	Log         logr.Logger
}

// +kubebuilder:rbac:groups=alpha.rudeboy.io,resources=pizzaorders,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=alpha.rudeboy.io,resources=pizzaorders/status,verbs=get;update;patch

func (r *PizzaOrderReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("pizzaorder", req.NamespacedName)

	pizzaOrder := &alphav1.PizzaOrder{}
	if err := r.Get(ctx, req.NamespacedName, pizzaOrder); err != nil {
		if kErr.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
	}

	defer func() {
		_ = r.Status().Update(ctx, pizzaOrder)
	}()

	if pizzaOrder.Status.Placed && !pizzaOrder.Status.Delivered {
		trackingURL, err := r.PizzaClient.GetTrackingUrl(pizzaOrder.Spec.Address.Phone)
		if err != nil {
			log.Error(err, "unable to get tracking URL")
			return ctrl.Result{}, nil
		}

		trackerStatus, err := r.PizzaClient.Track(trackingURL)
		if err != nil {
			log.Error(err, "unable to track order")
			return ctrl.Result{}, nil
		}

		tracker, err := convertTrackerStatus(trackerStatus)
		if err != nil {
			log.Error(err, "unable to convert tracker status to api type")
			return ctrl.Result{}, nil
		}

		if tracker.Delivered != "" {
			pizzaOrder.Status.Delivered = true
		}

		pizzaOrder.Status.Tracker = tracker
	} else {

		if pizzaOrder.Spec.Address == nil {
			return ctrl.Result{}, errors.New("address is empty")
		}

		apiAddress := convertAddress(pizzaOrder.Spec.Address)

		store, err := r.PizzaClient.GetNearestStore(apiAddress)
		if err != nil {
			return ctrl.Result{}, err
		}

		storeAddress := strings.Replace(store.Address, "\n", " ", -1)

		if !store.IsOpen {
			r.Log.Info("nearest store is not open for business", "address", storeAddress, "ID", store.StoreID)
			return ctrl.Result{}, nil
		}

		pizzaOrder.Status.Store = &alphav1.StoreStatus{
			ID:      store.StoreID,
			Address: storeAddress,
		}

		customer := pizzaOrder.Spec.Customer

		order := pizza.NewOrder().
			WithAddress(apiAddress).
			WithCustomerInfo(customer.FirstName, customer.LastName, customer.Email).
			WithPhoneNumber(strings.Replace(pizzaOrder.Spec.Address.Phone, "-", "", -1)).
			WithStoreID(store.StoreID)

		menu, err := r.PizzaClient.GetStoreMenu(store.StoreID)
		if err != nil {
			return ctrl.Result{}, err
		}

		for _, p := range pizzaOrder.Spec.Pizzas {
			product, err := r.identifyProduct(p, menu, store.StoreID)
			if err != nil {
				r.Log.Error(err, "error identifying product")
				continue
			}

			order.AddProduct(product)
		}

		// let's check for the fat 50% coupon
		order.AddCoupon(menu.GetFiftyPercentCouponCode())

		log.Info("validating order", "order", order)

		price, err := r.PizzaClient.ValidateOrder(order)
		if err != nil {
			log.Error(err, "unable to validate pizza")
			return ctrl.Result{}, nil
		}

		// because controller-gen does not support floats right now...
		pizzaOrder.Status.Price = strconv.FormatFloat(price, 'f', 2, 64)

		if pizzaOrder.Spec.PlaceOrder && !pizzaOrder.Status.Placed && pizzaOrder.Status.OrderID == "" {
			log.Info("placing order")
			payment, err := r.buildPaymentFromSecret(ctx, pizzaOrder.Spec.PaymentSecret.Name, pizzaOrder.GetNamespace())
			if err != nil {
				log.Error(err, "unable to format payment")
				return ctrl.Result{}, nil
			}

			payment.Amount = price

			order.Payments = []*pizza.Payment{payment}

			orderResponse, err := r.PizzaClient.PlaceOrder(order)
			if err != nil {
				log.Error(err, "unable to place order")
				return ctrl.Result{}, nil
			}

			// TODO: add better failure handling
			log.Info("order response", "order", orderResponse)

			pizzaOrder.Status.Placed = true
			pizzaOrder.Status.OrderID = orderResponse.OrderID
		}
	}

	return ctrl.Result{}, nil
}

func (r *PizzaOrderReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&alphav1.PizzaOrder{}).
		Complete(r)
}

func (r *PizzaOrderReconciler) identifyProduct(p *alphav1.Pizza, menu *pizza.Menu, storeID string) (*pizza.OrderProduct, error) {
	// get the size code
	var sizeCode string
	for _, size := range *menu.Sizes["Pizza"] {
		if strings.Contains(strings.ToLower(size.Name), strings.ToLower(p.Size)) {
			r.Log.Info("found size for pizza", "input", p.Size, "found", size.Name)
			sizeCode = size.Code
		}
	}

	if sizeCode == "" {
		return nil, errors.New("unable to find provided size in menu")
	}

	for _, variant := range menu.Products["S_PIZZA"].Variants {
		v, ok := menu.Variants[variant]
		if !ok {
			continue
		}

		// HANDTOSS is a nasty hardcode here. will need a way to map in styles
		// just means you can't order a THIN pizza for now :)
		if sizeCode == v.SizeCode && v.FlavorCode == "HANDTOSS" {
			updated, err := r.populateToppings(&pizza.OrderProduct{Code: v.Code, Qty: 1}, menu, p)
			if err != nil {
				r.Log.Error(err, "error populating toppings")
			}
			return updated, err
		}
	}

	return nil, errors.New("unable to identify product for provided pizza")
}

func (r *PizzaOrderReconciler) populateToppings(product *pizza.OrderProduct, menu *pizza.Menu, p *alphav1.Pizza) (*pizza.OrderProduct, error) {
	product.Options = make(map[string]*pizza.Option)

	// m*n RIP
	for _, topping := range p.Toppings {
		r.Log.Info("looking up topping", "topping", topping)
		for _, menuTopping := range *menu.Toppings["Pizza"] {
			if strings.Contains(strings.ToLower(menuTopping.Name), strings.ToLower(topping)) {
				r.Log.Info("found topping", "topping", topping, "code", menuTopping.Code)
				product.Options[menuTopping.Code] = &pizza.Option{"1/1": "1"}
			}
		}
	}

	if len(product.Options) < len(p.Toppings) {
		return product, errors.New("unable to identify all toppings")
	}

	return product, nil
}

func (r *PizzaOrderReconciler) buildPaymentFromSecret(ctx context.Context, secretName string, namespace string) (*pizza.Payment, error) {
	secret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: namespace, Name: secretName}, secret); err != nil {
		return nil, err
	}

	if secret.Data["CardType"] == nil {
		return nil, errors.New("CardType not found in secret")
	}

	if secret.Data["Number"] == nil {
		return nil, errors.New("Number not found in secret")
	}

	if secret.Data["Expiration"] == nil {
		return nil, errors.New("Expiration not found in secret")
	}

	if secret.Data["SecurityCode"] == nil {
		return nil, errors.New("SecurityCode not found in secret")
	}

	if secret.Data["PostalCode"] == nil {
		return nil, errors.New("PostalCode not found in secret")
	}

	return &pizza.Payment{
		Type:         "CreditCard",
		CardType:     strings.ToUpper(string(secret.Data["CardType"])),
		Number:       string(secret.Data["Number"]),
		Expiration:   string(secret.Data["Expiration"]),
		SecurityCode: string(secret.Data["SecurityCode"]),
		PostalCode:   string(secret.Data["PostalCode"]),
	}, nil
}

func convertAddress(in *alphav1.Address) *pizza.Address {
	return &pizza.Address{
		Street:     in.Street,
		City:       in.City,
		Region:     in.Region,
		PostalCode: in.PostalCode,
	}
}

func convertTrackerStatus(in *pizza.TrackerStatus) (*alphav1.Tracker, error) {
	return &alphav1.Tracker{
		Prep:           in.StartTime,
		Bake:           in.OvenTime,
		QualityCheck:   in.RackTime,
		OutForDelivery: in.RouteTime,
		Delivered:      in.DeliveryTime,
	}, nil
}
