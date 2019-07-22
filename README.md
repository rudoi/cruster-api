# :pizza: cruster-api: order Domino's with Kubernetes

This is a `kubebuilder`-generated Kubernetes API that knows how to order Domino's. See `config/samples/alpha_v1_pizzaorder.yaml` for a sample PizzaOrder.

Track your pizzas with `kubectl`!

```
> kubectl get pizzaorders
NAME                      PRICE   PREP                        BAKE                        QUALITY CHECK               DELIVERED
large-sausage-pineapple   19.07   2019-07-21T13:18:38-07:00   2019-07-21T13:20:01-07:00   2019-07-21T13:26:16-07:00   2019-07-21T13:42:11-07:00
```

The above order is :tada: [real](https://twitter.com/ndrewrudoi/status/1153056577508782082) :tada:.

Current limitations:

- US only
- Pizzas only (no pasta/wings/etc)

No releases have been cut, so clone this repo if you want to give it a shot. `make install && make run` should be enough to run locally!

**Please use at your own risk. I have placed exactly ONE real order with this so far.** :sweat_smile:

## notes on the PizzaOrder structure

`placeOrder` tells the controller whether or not it should actually place the order. This is a failsafe because I was nervous. :sunglasses:

`paymentSecret` refers to a Secret that must exist in the same namespace as the PizzaOrder itself. The controller expects the following keys:

- CardType (VISA/MASTERCARD/etc)
- Number (the CC number)
- Expiration (no slashes - ex. 0120 rather than 01/20)
- SecurityCode (the CVV on the back of the card)
- PostalCode (postal code of the billing address)

## Domino's SDK

This controller uses [pizza-go](https://github.com/rudoi/pizza-go), which I whipped up as part of this weekend project. :pizza:

## inspirations

- [terraform-provider-dominos](https://github.com/ndmckinley/terraform-provider-dominos)
- [node-dominos-pizza-api](https://github.com/RIAEvangelist/node-dominos-pizza-api)
