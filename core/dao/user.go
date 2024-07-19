package dao

import (
	"context"
	"core/models/entity"
	"core/repo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type UserDao struct {
	repo *repo.Manager
}

// 通过id查询用户是否存在
func (d *UserDao) FindUserByUid(ctx context.Context, uid string) (*entity.User, error) {
	db := d.repo.Mongo.Db.Collection("user")
	singleResult := db.FindOne(ctx, bson.D{
		{"uid", uid},
	})
	user := new(entity.User)
	err := singleResult.Decode(user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return user, nil
}

// 插入用户到数据库
func (d *UserDao) Insert(ctx context.Context, user *entity.User) error {
	db := d.repo.Mongo.Db.Collection("user")
	_, err := db.InsertOne(ctx, user)
	return err
}

// 创建用户的dao对象
func NewUserDao(m *repo.Manager) *UserDao {
	return &UserDao{
		repo: m,
	}
}

// 根据用户的id更新地址
func (d *UserDao) UpdateUserAddressByUid(ctx context.Context, user *entity.User) error {
	db := d.repo.Mongo.Db.Collection("user")
	_, err := db.UpdateOne(ctx, bson.M{
		"uid": user.Uid,
	}, bson.M{
		"$set": bson.M{
			"address":  user.Address,
			"location": user.Location,
		},
	})
	return err
}
