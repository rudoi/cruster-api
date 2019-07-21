package controllers

import (
	"context"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/rudoi/pizza-go/pkg/pizza"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	alphav1 "github.com/rudoi/cruster-api/api/v1"
)

var (
	r         *PizzaOrderReconciler
	req       ctrl.Request
	testOrder *alphav1.PizzaOrder
	ctx       context.Context
)

var _ = Describe("PizzaOrder reconciler", func() {
	BeforeEach(func() {
		ctx = context.Background()

		r = &PizzaOrderReconciler{
			Client: k8sClient,
			PizzaClient: &pizza.Client{
				Client: http.Client{Timeout: 10 * time.Second},
			},
			Log: logf.Log,
		}

		req = ctrl.Request{
			NamespacedName: types.NamespacedName{
				Namespace: "default",
				Name:      "test-pepperoni",
			},
		}

		testOrder = &alphav1.PizzaOrder{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "test-pepperoni",
			},
			Spec: alphav1.PizzaOrderSpec{
				Address: &alphav1.Address{
					Street:     "111 SW 5TH AVE",
					PostalCode: "97204",
					Phone:      "555-555-5555",
				},
				Pizzas: []*alphav1.Pizza{
					&alphav1.Pizza{
						Size:     "large",
						Toppings: []string{"pepperoni"},
					},
				},
			},
		}

		err := r.Create(ctx, testOrder)
		Expect(err).ToNot(HaveOccurred())
	})

	Context("when given a valid PizzaOrder", func() {
		It("should price out the order and update the object's status", func() {
			_, err := r.Reconcile(req)
			Expect(err).ToNot(HaveOccurred())

			pizzaOrder := &alphav1.PizzaOrder{}
			err = r.Get(ctx, req.NamespacedName, pizzaOrder)
			Expect(err).ToNot(HaveOccurred())
			Expect(pizzaOrder.Status.Price).ToNot(BeEmpty())
		})
	})
})
