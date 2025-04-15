package order

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	_ "order/elephas"
	"time"
)

func RegisterHandlers() {
	http.Handle("GET /order/{id}", http.HandlerFunc(getOrderHandler))
}

func getOrderHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	order := getOrder(id)
	w.Write([]byte(order.String()))
}

type order struct {
	id   int
	name string
}

func (o order) String() string {
	return fmt.Sprintf("id: %v; name: %v", o.id, o.name)
}

func getOrder(id string) order {
	ctx, cancle := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancle()
	var dsn = "postgres://postgres:postgres@localhost:5432/myorder"
	db, err := sql.Open("elephas", dsn)
	if err != nil {
		return order{}
	}
	var o order
	db.Ping()
	err = db.QueryRowContext(ctx, "select id, name from order_table where id = ?", id).Scan(&o.id, &o.name)
	if err != nil {
		log.Println(err)
		return order{}
	}
	return o
}
