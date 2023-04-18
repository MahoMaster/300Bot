package model

type Group struct {
	Id              int     `json:"id"`
	Group_id        float64 `json:"group_id"`
	Manager         float64 `json:"manager"`
	Gpt_personality string  `json:"gpt_personality"`
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

type User struct {
	Id                int    `json:"id,omitempty"`
	Qq                string `json:"qq,omitempty"`
	Is_ban            int    `json:"is_ban,omitempty"`
	Imgbackground_set int    `json:"imgbackground_set,omitempty"`
	Check_in          int64  `json:"check_in,omitempty"`
	Points            int    `json:"points,omitempty"`
	Last_chatgpt      string `json:"last_chatgpt"`
	Use_tokens        int    `json:"use_tokens"`
	Gpt_personality   string `json:"gpt_personality"`
	Gpt_use_person    int    `json:"gpt_use_person"`
}

type GPTPersonality struct {
	Id              string `json:"id"`
	Gpt_personality string `json:"gpt_personality"`
}

type UserGPTSetting struct {
	Gpt_use_person int    `json:"gpt_use_person"`
	Last_chatgpt   string `json:"last_chatgpt"`
	Is_ban         int    `json:"is_ban,omitempty"`
}

type QQFriend struct {
	User_id  float64 `json:"user_id"`
	Nickname string  `json:"nickName"`
	Remark   string  `json:"remark"`
}
