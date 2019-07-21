# :pizza: cruster-api: order Domino's with Kubernetes

This is a `kubebuilder`-generated Kubernetes API that knows how to order Domino's. See `config/samples/alpha_v1_pizzaorder.yaml` for a sample PizzaOrder. 

Current limitations:
- US only
- Pizzas only (no pasta/wings/etc)

No releases have been cut, so clone this repo if you want to give it a shot. `make install && make run` should be enough to run locally!


__Please use at your own risk. I have placed exactly ONE real order with this so far.__ :sweat_smile:

## notes on the PizzaOrder structure

`placeOrder` tells the controller whether or not it should actually place the order. This is a failsafe because I was nervous. :sunglasses:

`paymentSecret` refers to a Secret that must exist in the same namespace as the PizzaOrder itself. The controller expects the following keys:

- CardType (VISA/MASTERCARD/etc)
- Number (the CC number)
- Experiation (no slashes - ex. 0120 rather than 01/20)
- SecurityCode (the CVV on the back of the card)
- PostalCode (postal code of the billing address)