package services

import (
	"context"
	"github.com/PandaXGO/PandaKit/biz"
	"pandax/apps/device/entity"
	"pandax/pkg/global"
	"pandax/pkg/tool"
	"time"
)

type (
	DeviceModel interface {
		Insert(data entity.Device) *entity.Device
		FindOne(id string) *entity.DeviceRes
		FindListPage(page, pageSize int, data entity.Device) (*[]entity.DeviceRes, int64)
		FindList(data entity.Device) *[]entity.DeviceRes
		Update(data entity.Device) *entity.Device
		UpdateStatus(id, linkStatus string)
		Delete(ids []string)
	}

	deviceModelImpl struct {
		table string
	}
)

var DeviceModelDao DeviceModel = &deviceModelImpl{
	table: `devices`,
}

func (m *deviceModelImpl) Insert(data entity.Device) *entity.Device {
	//1 检查设备名称是否存在
	list := m.FindList(entity.Device{Name: data.Name})
	biz.IsTrue(list != nil && len(*list) == 0, "设备名称已经存在")
	//2 创建认证TOKEN IOTHUB使用
	etoken := getDeviceToken(&data)
	if data.DeviceType != global.GATEWAYS && data.DeviceType != global.MONITOR {
		data.Token = etoken.Token
	}
	//3 添加设备
	err := global.Db.Table(m.table).Create(&data).Error
	biz.ErrIsNil(err, "添加设备失败")
	return &data
}

func getDeviceToken(data *entity.Device) *tool.DeviceAuth {
	now := time.Now()
	etoken := &tool.DeviceAuth{
		DeviceId:   data.Id,
		User:       data.Owner,
		Name:       data.Name,
		DeviceType: data.DeviceType,
		ProductId:  data.Pid,
	}
	//设备有效期360天
	etoken.CreatedAt = now.Unix()
	etoken.ExpiredAt = now.Add(time.Hour * 24 * 365).Unix()
	if data.Token == "" {
		etoken.Token = etoken.MD5ID()
	} else {
		etoken.Token = data.Token
	}
	biz.ErrIsNil(global.RedisDb.Set(data.Id, etoken.GetMarshal(), time.Hour*24*365), "Redis 存储失败")
	return etoken
}

func (m *deviceModelImpl) FindOne(id string) *entity.DeviceRes {
	resData := new(entity.DeviceRes)
	db := global.Db.Table(m.table).Where("id = ?", id)
	err := db.First(resData).Preload("Product").Preload("DeviceGroup").Error
	biz.ErrIsNil(err, "查询设备失败")
	return resData
}

func (m *deviceModelImpl) FindListPage(page, pageSize int, data entity.Device) (*[]entity.DeviceRes, int64) {
	list := make([]entity.DeviceRes, 0)
	var total int64 = 0
	offset := pageSize * (page - 1)
	db := global.Db.Table(m.table)
	// 此处填写 where参数判断
	if data.Alias != "" {
		db = db.Where("alias = ?", data.Alias)
	}
	if data.Gid != "" {
		db = db.Where("gid = ?", data.Gid)
	}
	if data.OrgId != "" {
		db = db.Where("org_id = ?", data.OrgId)
	}
	if data.Name != "" {
		db = db.Where("name like ?", "%"+data.Name+"%")
	}
	if data.Owner != "" {
		db = db.Where("owner = ?", data.Owner)
	}
	if data.Status != "" {
		db = db.Where("status = ?", data.Status)
	}
	if data.LinkStatus != "" {
		db = db.Where("Link_status = ?", data.LinkStatus)
	}
	if data.Pid != "" {
		db = db.Where("pid = ?", data.Pid)
	}
	if data.ParentId != "" {
		db = db.Where("parent_id = ?", data.ParentId)
	}
	err := db.Count(&total).Error
	err = db.Order("create_time").Preload("Product").Preload("DeviceGroup").Limit(pageSize).Offset(offset).Find(&list).Error
	biz.ErrIsNil(err, "查询设备分页列表失败")
	return &list, total
}

func (m *deviceModelImpl) FindList(data entity.Device) *[]entity.DeviceRes {
	list := make([]entity.DeviceRes, 0)
	db := global.Db.Table(m.table)
	// 此处填写 where参数判断
	if data.Alias != "" {
		db = db.Where("alias = ?", data.Alias)
	}
	if data.Gid != "" {
		db = db.Where("gid = ?", data.Gid)
	}
	if data.OrgId != "" {
		db = db.Where("org_id = ?", data.OrgId)
	}
	if data.Name != "" {
		db = db.Where("name like ?", "%"+data.Name+"%")
	}
	if data.Owner != "" {
		db = db.Where("owner = ?", data.Owner)
	}
	if data.DeviceType != "" {
		db = db.Where("device_type = ?", data.DeviceType)
	}
	if data.Status != "" {
		db = db.Where("status = ?", data.Status)
	}
	if data.LinkStatus != "" {
		db = db.Where("Link_status = ?", data.LinkStatus)
	}
	if data.Pid != "" {
		db = db.Where("pid = ?", data.Pid)
	}
	if data.ParentId != "" {
		db = db.Where("parent_id = ?", data.ParentId)
	}
	db.Preload("Product").Preload("DeviceGroup")
	biz.ErrIsNil(db.Order("create_time").Find(&list).Error, "查询设备列表失败")
	return &list
}

func (m *deviceModelImpl) Update(data entity.Device) *entity.Device {
	getDeviceToken(&data)
	biz.ErrIsNil(global.Db.Table(m.table).Updates(&data).Error, "修改设备失败")
	return &data
}
func (m *deviceModelImpl) UpdateStatus(id, linkStatus string) {
	global.Db.Table(m.table).Where("id", id).Update("link_status", linkStatus).Update("last_time", time.Now())
}

func (m *deviceModelImpl) Delete(ids []string) {
	biz.ErrIsNil(global.Db.Table(m.table).Delete(&entity.Device{}, "id in (?)", ids).Error, "删除设备失败")
	for _, id := range ids {
		// 删除所有缓存
		global.RedisDb.Del(context.Background(), id)
	}
}
