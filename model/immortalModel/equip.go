package immortalModel

func GetEquipDetail(eid int, hasDetail int) (Equip, error) {
	var equip Equip
	r := db.Table("equip").Where("id=?", eid).First(&equip)
	if r.Error != nil {
		return equip, r.Error
	}
	if hasDetail != 0 {
		var entry = make([]Equip_entry, 0)
		r = db.Table("equip_entry").Where("eid=?", eid).Find(&entry)
		if r.Error != nil {
			return equip, r.Error
		}
		equip.Entry = entry
	}

	return equip, nil
}
