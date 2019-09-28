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

package rest

import (
	"github.com/gin-gonic/gin"
	"github.com/gitbitex/gitbitex-spot/service"
	"net/http"
	"time"
)

// POST /users
func SignUp(ctx *gin.Context) {
	var request SignUpRequest
	err := ctx.BindJSON(&request)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, newMessageVo(err))
		return
	}

	_, err = service.CreateUser(request.Email, request.Password)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, newMessageVo(err))
		return
	}
	ctx.JSON(http.StatusOK, nil)
}

// POST /users/accessToken
func SignIn(ctx *gin.Context) {
	var request SignUpRequest
	err := ctx.BindJSON(&request)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, newMessageVo(err))
		return
	}

	token, err := service.RefreshAccessToken(request.Email, request.Password)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, newMessageVo(err))
		return
	}

	ctx.SetCookie("accessToken", token, 7*24*60*60, "/", "*", false, false)
	ctx.JSON(http.StatusOK, token)
}

// POST /users/token
func GetToken(ctx *gin.Context) {
	var request SignUpRequest
	err := ctx.BindJSON(&request)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, newMessageVo(err))
		return
	}

	token, err := service.RefreshAccessToken(request.Email, request.Password)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, newMessageVo(err))
		return
	}
	ctx.JSON(http.StatusOK, token)
}

// POST /users/password
func ChangePassword(ctx *gin.Context) {
	var req changePasswordRequest
	err := ctx.BindJSON(&req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, newMessageVo(err))
		return
	}

	// check old password
	_, err = service.GetUserByPassword(GetCurrentUser(ctx).Email, req.OldPassword)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, newMessageVo(err))
		return
	}

	// change password
	err = service.ChangePassword(GetCurrentUser(ctx).Email, req.NewPassword)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, newMessageVo(err))
		return
	}

	ctx.JSON(http.StatusOK, nil)
}

// DELETE /users/accessToken
func SignOut(ctx *gin.Context) {
	ctx.SetCookie("accessToken", "", -1, "/", "*", false, false)
	ctx.JSON(http.StatusOK, nil)
}

// GET /users/self
func GetUsersSelf(ctx *gin.Context) {
	user := GetCurrentUser(ctx)
	if user == nil {
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	userVo := &userVo{
		Id:           user.Email,
		Email:        user.Email,
		Name:         user.Email,
		ProfilePhoto: "https://cdn.onlinewebfonts.com/svg/img_139247.png",
		IsBand:       false,
		CreatedAt:    user.CreatedAt.Format(time.RFC3339),
	}

	ctx.JSON(http.StatusOK, userVo)
}
