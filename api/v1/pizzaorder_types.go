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

package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Pizza struct {
	// +kubebuilder:validation:Enum=small;medium;large
	Size string `json:"size"`

	Toppings []string `json:"toppings"`
}

type Customer struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
}

type Address struct {
	Street string `json:"street"`
	City   string `json:"city"`

	// +kubebuilder:validation:MaxLength=2
	Region string `json:"region"`

	PostalCode string `json:"postalCode"`

	// +kubebuilder:validation:Pattern="[2-9]\\d{9}$"
	Phone string `json:"phone"`
}

type StoreStatus struct {
	ID      string `json:"id,omitempty"`
	Address string `json:"address,omitempty"`
}

type Tracker struct {
	Prep           string `json:"prep,omitempty"`
	Bake           string `json:"bake,omitempty"`
	QualityCheck   string `json:"qualityCheck,omitempty"`
	OutForDelivery string `json:"outForDelivery,omitempty"`
	Delivered      string `json:"delivered,omitempty"`
}

// PizzaOrderSpec defines the desired state of PizzaOrder
type PizzaOrderSpec struct {
	PlaceOrder    bool                        `json:"placeOrder"`
	Address       *Address                    `json:"address"`
	Customer      *Customer                   `json:"customer"`
	PaymentSecret corev1.LocalObjectReference `json:"paymentSecret,omitempty"`

	// +kubebuilder:validation:MinItems=1
	Pizzas []*Pizza `json:"pizzas"`
}

// PizzaOrderStatus defines the observed state of PizzaOrder
type PizzaOrderStatus struct {
	OrderID   string       `json:"orderID,omitempty"`
	Price     string       `json:"price,omitempty"`
	Placed    bool         `json:"placed,omitempty"`
	Delivered bool         `json:"delivered,omitempty"`
	Store     *StoreStatus `json:"store,omitempty"`
	Tracker   *Tracker     `json:"tracker,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=pizzaorders,shortName=pz
// +kubebuilder:printcolumn:name="price",type="string",JSONPath=".status.price",description="order price"
// +kubebuilder:printcolumn:name="prep",type="string",JSONPath=".status.tracker.prep",description="prep start time"
// +kubebuilder:printcolumn:name="bake",type="string",JSONPath=".status.tracker.bake",description="bake start time"
// +kubebuilder:printcolumn:name="quality check",type="string",JSONPath=".status.tracker.qualityCheck",description="quality check start time"
// +kubebuilder:printcolumn:name="delivered",type="string",JSONPath=".status.tracker.delivered",description="delivered time"

// PizzaOrder is the Schema for the pizzaorders API
type PizzaOrder struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PizzaOrderSpec   `json:"spec,omitempty"`
	Status PizzaOrderStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PizzaOrderList contains a list of PizzaOrder
type PizzaOrderList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PizzaOrder `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PizzaOrder{}, &PizzaOrderList{})
}
