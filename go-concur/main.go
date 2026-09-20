package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

type Order struct {
	ID     int
	Status string
	mu     sync.Mutex
}

var (
	totalUpdates int
	updateMutex  sync.Mutex
)

// Concurrency is Dealing with a lot of things at the same time.
// Parallelism is Doing a lot of things at the same time

// WaitGroup - the issue here is when concurrent funcs try to access the same data
// mutex - mutual exclusion - will solve that problem
// Mutex is a locking mechanism that insures only one go routine can access a specific
// section of your code. It is used to prevent race conditions
func main() {
	// wg := sync.WaitGroup{}
	var wg sync.WaitGroup
	wg.Add(2)

	orders := generateOrders(10)

	go func() {
		defer wg.Done()
		processOrders(orders)
	}()

	for range 3 {
		go func() {
			defer wg.Done()
			for _, order := range orders {
				updateOrderStatuses(order)
			}
		}()
	}

	wg.Wait()

	reportOrderStatus(orders)
	fmt.Println(totalUpdates)

	fmt.Println("All operations completed. Exiting")
}

func updateOrderStatuses(order *Order) {
	order.mu.Lock()

	time.Sleep(
		time.Duration(rand.Intn(300)) * time.Millisecond,
	)

	status := []string{
		"Processing", "Shipped", "Delivered",
	}[rand.Intn(3)]
	order.Status = status
	fmt.Printf("Updated order %d status: %s\n",
		order.ID, status,
	)

	order.mu.Unlock()

	updateMutex.Lock()
	defer updateMutex.Unlock()
	currentUpdates := totalUpdates
	time.Sleep(5 * time.Millisecond)
	totalUpdates = currentUpdates + 1
}

func processOrders(orders []*Order) {
	for _, order := range orders {
		time.Sleep(
			time.Duration(rand.Intn(500)) *
				time.Millisecond,
		)

		fmt.Println("Processing order", order.ID)
	}
}

// generate n amount of orders
func generateOrders(count int) []*Order {
	orders := make([]*Order, count)

	// for i := 0; i < count; i++ {
	// for loop can be modernized using range over int [default]
	for i := range count {
		orders[i] = &Order{
			ID:     i + 1,
			Status: "Pending",
		}
	}

	fmt.Println("Orders created!!")
	return orders
}

func reportOrderStatus(orders []*Order) {
	fmt.Println("\n--- Order Status Report ---")

	for _, order := range orders {
		fmt.Printf(
			"Order %d: %s\n",
			order.ID, order.Status,
		)
	}

	fmt.Println("--------------------------")
}
