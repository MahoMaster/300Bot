package immortal

import (
	"300Bot/model/immortalModel"
	"300Bot/send"
	"errors"
	"strconv"
)

func GetEquipDetail(eidStr string, msg map[string]interface{}) error {

	eid, err := strconv.Atoi(eidStr)
	if err != nil {
		return errors.New("参数错误")
	}
	equip, err := immortalModel.GetEquipDetail(eid, 1)
	if err != nil {
		return err
	}
	template := equip.Name + `:
	` + Root2SkillRootName(equip.Root) + Type2EquipTypeName(equip.Type) + `,
	` + equip.Intro + `
------------------------------`
	for _, item := range equip.Entry {
		template = template + `
	` + item.Content
	}
	send.SendGroupPost(msg["group_id"].(float64), template)
	return nil
}

func BuyEquip(qq string, eidStr string, msg map[string]interface{}) error {
	eid, err := strconv.Atoi(eidStr)
	if err != nil {
		return errors.New("参数错误")
	}

	sa, err := immortalModel.GetAdminShopOne(3, eid, 0)
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

	us, _ := immortalModel.GetUserEquipOne(u.Id, eid, 0)
	if us.Eid != 0 {
		return errors.New("你都有了还买个锤子")
	}

	err = immortalModel.BuyEquip(u.Id, sa.Price, eid)
	if err != nil {
		return err
	}
	send.SendGroupPost(msg["group_id"].(float64), "购买成功")
	return nil
}
