// Copyright 2015-present Oursky Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package skydb

import (
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/skygeario/skygear-server/pkg/server/utils"
	"github.com/skygeario/skygear-server/pkg/server/uuid"
)

// ProviderInfo represents the dictionary of authenticated principal ID => authData.
//
// For example, a AuthInfo connected with a Facebook account might
// look like this:
//
//   {
//     "com.facebook:46709394": {
//       "accessToken": "someAccessToken",
//       "expiredAt": "2015-02-26T20:05:48",
//       "facebookID": "46709394"
//     }
//   }
//
// It is assumed that the Facebook AuthProvider has "com.facebook" as
// provider name and "46709394" as the authenticated Facebook account ID.
type ProviderInfo map[string]map[string]interface{}

// AuthInfo contains a user's information for authentication purpose
type AuthInfo struct {
	ID              string       `json:"_id"`
	HashedPassword  []byte       `json:"password,omitempty"`
	Roles           []string     `json:"roles,omitempty"`
	ProviderInfo    ProviderInfo `json:"provider_info,omitempty"` // auth data for alternative methods
	TokenValidSince *time.Time   `json:"token_valid_since,omitempty"`
	LastSeenAt      *time.Time   `json:"last_seen_at,omitempty"`
}

// AuthData contains the unique authentication data of a user
// e.g.: {"username": "userA", "email": "userA@abc.com"}
type AuthData struct {
	data map[string]interface{}
	keys [][]string
}

func NewAuthData(data map[string]interface{}, authRecordKeys [][]string) AuthData {
	return AuthData{
		data: data,
		keys: authRecordKeys,
	}
}

func (a AuthData) allKeys() []string {
	keyMap := map[string]bool{}
	for _, keys := range a.keys {
		for _, key := range keys {
			keyMap[key] = true
		}
	}

	keys := []string{}
	for k := range keyMap {
		keys = append(keys, k)
	}

	return keys
}

func (a AuthData) usingKeys() []string {
	for _, ks := range a.keys {
		count := 0
		for _, k := range ks {
			for dk := range a.data {
				if k == dk && !a.isFieldEmpty(dk) {
					count = count + 1
				}
			}
		}

		if len(ks) == count {
			return ks
		}
	}

	return []string{}
}

func (a AuthData) GetData() map[string]interface{} {
	c := map[string]interface{}{}
	for k, v := range a.data {
		c[k] = v
	}
	return c
}

func (a AuthData) MakeEqualPredicate() Predicate {
	appendEqualPredicateForKey := func(predicates []interface{}, key string) []interface{} {
		if a.data[key] == nil {
			return predicates
		}

		return append(predicates, Predicate{
			Operator: Equal,
			Children: []interface{}{
				Expression{Type: KeyPath, Value: key},
				Expression{Type: Literal, Value: a.data[key]},
			},
		})
	}

	predicates := []interface{}{}
	for _, k := range a.usingKeys() {
		predicates = appendEqualPredicateForKey(predicates, k)
	}

	return Predicate{
		Operator: And,
		Children: predicates,
	}
}

func (a *AuthData) UpdateFromRecordData(data Data) {
	for _, k := range a.allKeys() {
		a.data[k] = data[k]
	}
}

func (a AuthData) IsValid() bool {
	for dk := range a.data {
		found := false
		for _, k := range a.allKeys() {
			if dk == k {
				found = true
				break
			}
		}

		if !found {
			return false
		}
	}

	return len(a.usingKeys()) > 0
}

// IsEmpty would return true if
//
// 1. no entries or
// 2. all values are null
func (a AuthData) IsEmpty() bool {
	if len(a.data) == 0 {
		return true
	}

	for k := range a.data {
		if !a.isFieldEmpty(k) {
			return false
		}
	}

	return true
}

func (a AuthData) isFieldEmpty(key string) bool {
	return a.data[key] == nil
}

// NewAuthInfo returns a new AuthInfo with specified password.
// An UUID4 ID will be generated by the system as unique identifier
func NewAuthInfo(password string) AuthInfo {
	info := AuthInfo{
		ID: uuid.New(),
	}
	info.SetPassword(password)

	return info
}

// NewAnonymousAuthInfo returns an anonymous AuthInfo, which has
// no Password.
func NewAnonymousAuthInfo() AuthInfo {
	return AuthInfo{
		ID: uuid.New(),
	}
}

// NewProviderInfoAuthInfo returns an AuthInfo provided by a AuthProvider,
// which has no Password.
func NewProviderInfoAuthInfo(principalID string, authData map[string]interface{}) AuthInfo {
	return AuthInfo{
		ID: uuid.New(),
		ProviderInfo: ProviderInfo(map[string]map[string]interface{}{
			principalID: authData,
		}),
	}
}

// SetPassword sets the HashedPassword with the password specified
func (info *AuthInfo) SetPassword(password string) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		panic("authinfo: Failed to hash password")
	}

	info.HashedPassword = hashedPassword

	// Changing the password will also update the time before which issued
	// access token should be invalidated.
	timeNow := time.Now().UTC()
	info.TokenValidSince = &timeNow
}

// IsSamePassword determines whether the specified password is the same
// password as where the HashedPassword is generated from
func (info AuthInfo) IsSamePassword(password string) bool {
	return bcrypt.CompareHashAndPassword(info.HashedPassword, []byte(password)) == nil
}

// SetProviderInfoData sets the auth data to the specified principal.
func (info *AuthInfo) SetProviderInfoData(principalID string, authData map[string]interface{}) {
	if info.ProviderInfo == nil {
		info.ProviderInfo = make(map[string]map[string]interface{})
	}
	info.ProviderInfo[principalID] = authData
}

// HasAnyRoles return true if authinfo belongs to one of the supplied roles
func (info *AuthInfo) HasAnyRoles(roles []string) bool {
	return utils.StringSliceContainAny(info.Roles, roles)
}

// HasAllRoles return true if authinfo has all roles supplied
func (info *AuthInfo) HasAllRoles(roles []string) bool {
	return utils.StringSliceContainAll(info.Roles, roles)
}

// GetProviderInfoData gets the auth data for the specified principal.
func (info *AuthInfo) GetProviderInfoData(principalID string) map[string]interface{} {
	if info.ProviderInfo == nil {
		return nil
	}
	value, _ := info.ProviderInfo[principalID]
	return value
}

// RemoveProviderInfoData remove the auth data for the specified principal.
func (info *AuthInfo) RemoveProviderInfoData(principalID string) {
	if info.ProviderInfo != nil {
		delete(info.ProviderInfo, principalID)
	}
}
