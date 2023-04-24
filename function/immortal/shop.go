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
	类型:` + Type2SkillTypeName(item.Skill.Type) + `,
	价格:` + Number2String(item.Price) + `灵石,
	` + item.Skill.Intro + `
------------------------------`
	}
	send.SendGroupPost(msg["group_id"].(float64), template)
	return nil
}
func GetEquipAdminShop(qq string, msg map[string]interface{}, name string, page int) error {
	mods, err := immortalModel.GetAdminShop(3, name, page)
	if err != nil {
		return err
	}
	template := `系统装备商城:
------------------------------`
	for _, item := range mods {
		template = template + `
	物品` + Number2String(item.Gid) + `:` + item.Equip.Name + `,
	类型:` + Type2EquipTypeName(item.Equip.Type) + `,
	价格:` + Number2String(item.Price) + `灵石,
	` + item.Equip.Intro + `
------------------------------`
	}
	send.SendGroupPost(msg["group_id"].(float64), template)
	return nil
}
