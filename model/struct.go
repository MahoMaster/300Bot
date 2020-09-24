package model

type Group struct {
	Id       int     `json:"id"`
	Group_id float64 `json:"group_id"`
	Manager  float64 `json:"manager"`
}

type At struct {
	Id      int    `json:"id"`
	Keyword string `json:"keyword"`
	Reply   string `json:"reply"`
}
