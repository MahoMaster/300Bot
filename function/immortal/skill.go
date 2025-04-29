package immortal

import (
	"300Bot/model/immortalModel"
	"300Bot/send"
	"errors"
	"strconv"
)

func GetSkillDetail(sidStr string, msg map[string]interface{}) error {

	sid, err := strconv.Atoi(sidStr)
	if err != nil {
		return errors.New("参数错误")
	}
	skill, err := immortalModel.GetSkillDetail(sid, 1)
	if err != nil {
		return err
	}
	template := skill.Name + `:
	` + Root2SkillRootName(skill.Root) + Type2SkillTypeName(skill.Type) + `,
	` + skill.Intro + `
------------------------------`
	for _, item := range skill.Entry {
		template = template + `
	` + item.Content
	}
	send.SendGroupPost(msg["group_id"].(float64), template)
	return nil
}

func BuySkill(qq string, sidStr string, msg map[string]interface{}) error {
	sid, err := strconv.Atoi(sidStr)
	if err != nil {
		return errors.New("参数错误")
	}

	sa, err := immortalModel.GetAdminShopOne(1, sid, 0)
	if err != nil {
		return err
	}

	u, err := immortalModel.GetUserInfoByQQ(qq)

	if err != nil {
		return err
	}
	uc, _, _, err := immortalModel.GetUserCultivateById(u.Id)
	if err != nil {
		return err
	}
	if uc.Stone < sa.Price {
		return errors.New("灵石不够啊衰仔")
	}

	us, _ := immortalModel.GetUserSkillOne(u.Id, sid, 0)
	if us.Sid != 0 {
		return errors.New("你都有了还买个锤子")
	}

	err = immortalModel.BuySkill(u.Id, sa.Price, sid)
	if err != nil {
		return err
	}
	send.SendGroupPost(msg["group_id"].(float64), "购买成功")
	return nil
}
