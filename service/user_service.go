// Copyright 2019 GitBitEx.com
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package service

import (
	"crypto/md5"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/gitbitex/gitbitex-spot/conf"
	"github.com/gitbitex/gitbitex-spot/models"
	"github.com/gitbitex/gitbitex-spot/models/mysql"
	"github.com/pkg/errors"
	"time"
)

func CreateUser(email, password string) (*models.User, error) {
	if len(password) < 6 {
		return nil, errors.New("password must be of minimum 6 characters length")
	}
	user, err := GetUserByEmail(email)
	if err != nil {
		return nil, err
	}
	if user != nil {
		return nil, errors.New("email address is already registered")
	}

	user = &models.User{
		Email:        email,
		PasswordHash: encryptPassword(password),
	}
	return user, mysql.SharedStore().AddUser(user)
}

func RefreshAccessToken(email, password string) (string, error) {
	user, err := GetUserByEmail(email)
	if err != nil {
		return "", err
	}
	if user == nil {
		return "", errors.New("email not found or password error")
	}
	if user.PasswordHash != encryptPassword(password) {
		return "", errors.New("email not found or password error")
	}

	claim := jwt.MapClaims{
		"id":           user.Id,
		"email":        user.Email,
		"passwordHash": user.PasswordHash,
		"expiredAt":    time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claim)
	return token.SignedString([]byte(conf.GetConfig().JwtSecret))
}

func CheckToken(tokenStr string) (*models.User, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return []byte(conf.GetConfig().JwtSecret), nil
	})
	if err != nil {
		return nil, err
	}
	claim, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("cannot convert claim to MapClaims")
	}
	if !token.Valid {
		return nil, errors.New("token is invalid")
	}

	emailVal, found := claim["email"]
	if !found {
		return nil, errors.New("bad token")
	}
	email := emailVal.(string)

	passwordHashVal, found := claim["passwordHash"]
	if !found {
		return nil, errors.New("bad token")
	}
	passwordHash := passwordHashVal.(string)

	user, err := GetUserByEmail(email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("bad token")
	}
	if user.PasswordHash != passwordHash {
		return nil, errors.New("bad token")
	}
	return user, nil
}

func ChangePassword(email, newPassword string) error {
	user, err := GetUserByEmail(email)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("user not found")
	}
	user.PasswordHash = encryptPassword(newPassword)
	return mysql.SharedStore().UpdateUser(user)
}

func GetUserByEmail(email string) (*models.User, error) {
	return mysql.SharedStore().GetUserByEmail(email)
}

func GetUserByPassword(email, password string) (*models.User, error) {
	user, err := GetUserByEmail(email)
	if err != nil {
		return nil, err
	}
	if user == nil || user.PasswordHash != encryptPassword(password) {
		return nil, errors.New("user not found or password incorrect")
	}
	return user, nil
}

func encryptPassword(password string) string {
	hash := md5.Sum([]byte(password))
	return fmt.Sprintf("%x", hash)
}
