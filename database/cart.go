package database

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/akhil/ecommerce-yt/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	ErrCantFindProduct    = errors.New("can't find product")
	ErrCantDecodeProducts = errors.New("can't decode products")
	ErrUserIDIsNotValid   = errors.New("user ID is not valid")
	ErrCantUpdateUser     = errors.New("cannot add product to cart")
	ErrCantRemoveItem     = errors.New("cannot remove item from cart")
	ErrCantGetItem        = errors.New("cannot get item from cart")
	ErrCantBuyCartItem    = errors.New("cannot complete the purchase")
)

func AddProductToCart(ctx context.Context, prodCollection, userCollection *mongo.Collection, productID primitive.ObjectID, userID string) error {
	searchfromdb, err := prodCollection.Find(ctx, bson.M{"_id": productID})
	if err != nil {
		log.Println(err)
		return ErrCantFindProduct
	}

	var productcart []models.ProductUser
	if err := searchfromdb.All(ctx, &productcart); err != nil {
		log.Println(err)
		return ErrCantDecodeProducts
	}

	id, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		log.Println(err)
		return ErrUserIDIsNotValid
	}

	filter := bson.M{"_id": id}
	update := bson.M{
		"$push": bson.M{
			"usercart": bson.M{
				"$each": productcart,
			},
		},
	}

	_, err = userCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		log.Println(err)
		return ErrCantUpdateUser
	}

	return nil
}

func RemoveCartItem(ctx context.Context, prodCollection, userCollection *mongo.Collection, productID primitive.ObjectID, userID string) error {
	id, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		log.Println(err)
		return ErrUserIDIsNotValid
	}

	filter := bson.M{"_id": id}
	update := bson.M{"$pull": bson.M{"usercart": bson.M{"_id": productID}}}

	_, err = userCollection.UpdateOne(ctx, filter, update) // ❗ Use UpdateOne here
	if err != nil {
		log.Println(err)
		return ErrCantRemoveItem
	}

	return nil
}

func BuyItemFromCart(ctx context.Context, userCollection *mongo.Collection, userID string) error {
	id, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		log.Println(err)
		return ErrUserIDIsNotValid
	}

	var getcartitems models.User
	var ordercart models.Order
	ordercart.Order_ID = primitive.NewObjectID()
	ordercart.Orderered_At = time.Now()
	ordercart.Order_Cart = make([]models.ProductUser, 0)
	ordercart.Payment_Method.COD = true

	unwind := bson.D{{Key: "$unwind", Value: "$usercart"}}
	grouping := bson.D{{
		Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$_id"},
			{Key: "total", Value: bson.D{{Key: "$sum", Value: "$usercart.price"}}},
		},
	}}

	cursor, err := userCollection.Aggregate(ctx, mongo.Pipeline{unwind, grouping})
	if err != nil {
		log.Println("Aggregation error:", err)
		return ErrCantBuyCartItem
	}
	defer cursor.Close(ctx)

	var result []bson.M
	if err := cursor.All(ctx, &result); err != nil {
		log.Println("Cursor decode error:", err)
		return ErrCantBuyCartItem
	}

	var totalPrice int32
	if len(result) > 0 {
		if price, ok := result[0]["total"].(int32); ok {
			totalPrice = price
		} else {
			log.Println("Failed to assert total price as int32")
		}
	}

	ordercart.Price = int(totalPrice)

	filter := bson.M{"_id": id}
	update := bson.M{"$push": bson.M{"orders": ordercart}}
	if _, err := userCollection.UpdateOne(ctx, filter, update); err != nil {
		log.Println(err)
		return ErrCantBuyCartItem
	}

	if err := userCollection.FindOne(ctx, filter).Decode(&getcartitems); err != nil {
		log.Println(err)
		return ErrCantBuyCartItem
	}

	// Add products from cart to the order list
	update2 := bson.M{
		"$push": bson.M{
			"orders.$[].order_list": bson.M{
				"$each": getcartitems.UserCart,
			},
		},
	}
	if _, err := userCollection.UpdateOne(ctx, filter, update2); err != nil {
		log.Println(err)
	}

	// Empty user cart after order
	emptyCart := make([]models.ProductUser, 0)
	clearCart := bson.M{"$set": bson.M{"usercart": emptyCart}}

	if _, err := userCollection.UpdateOne(ctx, filter, clearCart); err != nil {
		log.Println(err)
		return ErrCantBuyCartItem
	}

	return nil
}

func InstantBuyer(ctx context.Context, prodCollection, userCollection *mongo.Collection, productID primitive.ObjectID, userID string) error {
	id, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		log.Println(err)
		return ErrUserIDIsNotValid
	}

	var product models.ProductUser
	if err := prodCollection.FindOne(ctx, bson.M{"_id": productID}).Decode(&product); err != nil {
		log.Println(err)
		return ErrCantFindProduct
	}

	order := models.Order{
		Order_ID:        primitive.NewObjectID(),
		Orderered_At:    time.Now(),
		Order_Cart:      []models.ProductUser{product},
		Price:           product.Price,
		Payment_Method:  models.Payment{COD: true},
	}

	filter := bson.M{"_id": id}
	update := bson.M{"$push": bson.M{"orders": order}}

	if _, err := userCollection.UpdateOne(ctx, filter, update); err != nil {
		log.Println(err)
		return ErrCantBuyCartItem
	}

	// Add product to the most recent order's list
	update2 := bson.M{
		"$push": bson.M{
			"orders.$[].order_list": product,
		},
	}
	if _, err := userCollection.UpdateOne(ctx, filter, update2); err != nil {
		log.Println(err)
	}

	return nil
}
