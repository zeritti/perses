// Copyright 2021 The Perses Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package globalsecret

import (
	"fmt"

	"github.com/perses/perses/internal/api/interface/v1/globalsecret"
	"github.com/perses/perses/internal/api/shared"
	"github.com/perses/perses/internal/api/shared/crypto"
	databaseModel "github.com/perses/perses/internal/api/shared/database/model"
	"github.com/perses/perses/pkg/model/api"
	v1 "github.com/perses/perses/pkg/model/api/v1"
	"github.com/sirupsen/logrus"
)

type service struct {
	globalsecret.Service
	dao    globalsecret.DAO
	crypto crypto.Crypto
}

func NewService(dao globalsecret.DAO, crypto crypto.Crypto) globalsecret.Service {
	return &service{
		dao:    dao,
		crypto: crypto,
	}
}

func (s *service) Create(entity api.Entity) (interface{}, error) {
	if object, ok := entity.(*v1.GlobalSecret); ok {
		return s.create(object)
	}
	return nil, shared.HandleBadRequestError(fmt.Sprintf("wrong entity format, attempting GlobalSecret format, received '%T'", entity))
}

func (s *service) create(entity *v1.GlobalSecret) (*v1.PublicGlobalSecret, error) {
	// Update the time contains in the entity
	entity.Metadata.CreateNow()
	if err := s.crypto.Encrypt(&entity.Spec); err != nil {
		logrus.WithError(err).Errorf("unable to encrypt the secret spec")
		return nil, shared.InternalError
	}
	if err := s.dao.Create(entity); err != nil {
		return nil, err
	}
	return v1.NewPublicGlobalSecret(entity), nil
}

func (s *service) Update(entity api.Entity, parameters shared.Parameters) (interface{}, error) {
	if object, ok := entity.(*v1.GlobalSecret); ok {
		return s.update(object, parameters)
	}
	return nil, shared.HandleBadRequestError(fmt.Sprintf("wrong entity format, attempting GlobalSecret format, received '%T'", entity))
}

func (s *service) update(entity *v1.GlobalSecret, parameters shared.Parameters) (*v1.PublicGlobalSecret, error) {
	if entity.Metadata.Name != parameters.Name {
		logrus.Debugf("name in GlobalSecret %q and name from the http request: %q don't match", entity.Metadata.Name, parameters.Name)
		return nil, shared.HandleBadRequestError("metadata.name and the name in the http path request don't match")
	}
	// find the previous version of the GlobalSecret
	oldEntity, err := s.dao.Get(parameters.Name)
	if err != nil {
		return nil, err
	}
	entity.Metadata.Update(oldEntity.Metadata)

	if encryptErr := s.crypto.Encrypt(&entity.Spec); encryptErr != nil {
		logrus.WithError(encryptErr).Errorf("unable to encrypt the secret spec")
		return nil, shared.InternalError
	}
	if updateErr := s.dao.Update(entity); updateErr != nil {
		logrus.WithError(updateErr).Errorf("unable to perform the update of the GlobalSecret %q, something wrong with the database", entity.Metadata.Name)
		return nil, updateErr
	}
	return v1.NewPublicGlobalSecret(entity), nil
}

func (s *service) Delete(parameters shared.Parameters) error {
	return s.dao.Delete(parameters.Name)
}

func (s *service) Get(parameters shared.Parameters) (interface{}, error) {
	scrt, err := s.dao.Get(parameters.Name)
	if err != nil {
		return nil, err
	}
	return v1.NewPublicGlobalSecret(scrt), nil
}

func (s *service) List(q databaseModel.Query, _ shared.Parameters) (interface{}, error) {
	l, err := s.dao.List(q)
	if err != nil {
		return nil, err
	}
	result := make([]*v1.PublicGlobalSecret, 0, len(l))
	for _, scrt := range l {
		result = append(result, v1.NewPublicGlobalSecret(scrt))
	}
	return result, nil
}
