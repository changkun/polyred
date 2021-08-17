// Copyright 2021 Changkun Ou <changkun.de>. All rights reserved.
// Use of this source code is governed by a GPLv3 license that
// can be found in the LICENSE file.

package utils

// DefaultLimit is the default Conccurrent limit
const DefaultLimit = 100

// Limiter object
type Limiter struct {
	limit   int
	tickets chan struct{}
}

// NewConccurLimiter allocates a new ConccurLimiter
func NewLimiter(limit int) *Limiter {
	if limit <= 0 {
		limit = DefaultLimit
	}

	// allocate a limiter instance
	c := &Limiter{
		limit:   limit,
		tickets: make(chan struct{}, limit),
	}

	// allocate the tickets:
	for i := 0; i < c.limit; i++ {
		c.tickets <- struct{}{}
	}

	return c
}

// Execute adds a function to the execution queue.
// if num of go routines allocated by this instance is < limit
// launch a new go routine to execute job
// else wait until a go routine becomes available
func (c *Limiter) Execute(job func()) {
	ticket := <-c.tickets
	go func() {
		defer func() {
			c.tickets <- ticket
		}()
		job()
	}()
}

// Wait will block all the previously Executed jobs completed running.
// Note that calling the Wait function while keep calling Execute leads
// to un-desired race conditions
func (c *Limiter) Wait() {
	for i := 0; i < c.limit; i++ {
		<-c.tickets
	}

	// reset all tickets
	c.tickets = make(chan struct{}, c.limit)
	for i := 0; i < c.limit; i++ {
		c.tickets <- struct{}{}
	}
}
