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
	Need_at int    `json:"need_at"`
	Need_qq int    `json:"need_qq"`
}

type ImgBackground struct {
	Id           int     `json:"id,omitempty"`
	File_name    string  `json:"file_name,omitempty"`
	Y_start      int     `json:"y_start,omitempty"`
	X_start      int     `json:"x_start,omitempty"`
	X_end        int     `json:"x_end,omitempty"`
	Max_length   int     `json:"max_length,omitempty"`
	Ttf_name     string  `json:"ttf_name,omitempty"`
	Font_size    float64 `json:"font_size,omitempty"`
	Font_spacing float64 `json:"font_spacing,omitempty"`
	Rect_x       int     `json:"rect_x,omitempty"`
	Rect_y       int     `json:"rect_y,omitempty"`
}
