package immortal

import (
	"300Bot/model/immortalModel"
	"300Bot/send"
)

func GetSkillAdminShop(qq string, msg map[string]interface{}, name string, page int) error {
	mods, err := immortalModel.GetAdminShop(1, name, page)
	if err != nil {
		return err
	}
	template := `系统功法商城:
------------------------------`
	for _, item := range mods {
		template = template + `
	功法` + Number2String(item.Gid) + `:` + item.Skill.Name + `,
	价格:` + Number2String(item.Price) + `灵石,
	` + item.Skill.Intro + `
------------------------------`
	}
	send.SendGroupPost(msg["group_id"].(float64), template)
	return nil
}
