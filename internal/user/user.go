// Package user respresents user entity and operations that can be made with this kind of entities
package user

type UserRequest struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

type User struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Age  int    `json:"age"`
}
