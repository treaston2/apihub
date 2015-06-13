package mongore

import (
	"fmt"
	"time"

	"github.com/backstage/backstage/account"
	"github.com/backstage/backstage/db"
	"github.com/backstage/backstage/errors"
	. "github.com/backstage/backstage/log"
	"github.com/fatih/structs"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type Mongore struct {
	config Config
}

func New(config Config) account.Storable {
	return &Mongore{config: config}
}

func (m *Mongore) UpsertUser(u account.User) error {
	_, err := m.Users().Upsert(bson.M{"email": u.Email}, u)

	if err != nil {
		Logger.Warn(err.Error())
	}

	return err
}

func (m *Mongore) DeleteUser(u account.User) error {
	err := m.Users().Remove(u)

	if err == mgo.ErrNotFound {
		return errors.NewNotFoundErrorNEW(errors.ErrUserNotFound)
	}
	if err != nil {
		Logger.Warn(err.Error())
	}

	return err
}

func (m *Mongore) FindUserByEmail(email string) (account.User, error) {
	user := account.User{}
	err := m.Users().Find(bson.M{"email": email}).One(&user)

	if err == mgo.ErrNotFound {
		return account.User{}, errors.NewNotFoundErrorNEW(errors.ErrUserNotFound)
	}
	if err != nil {
		Logger.Warn(err.Error())
	}

	return user, err
}

func (m *Mongore) UserTeams(user account.User) ([]account.Team, error) {
	teams := []account.Team{}
	err := m.Teams().Find(bson.M{"users": bson.M{"$in": []string{user.Email}}}).All(&teams)
	return teams, err
}

func (m *Mongore) UpsertTeam(t account.Team) error {
	_, err := m.Teams().Upsert(bson.M{"alias": t.Alias}, t)

	if err != nil {
		Logger.Warn(err.Error())
	}

	return err
}

func (m *Mongore) DeleteTeam(t account.Team) error {
	err := m.Teams().Remove(t)

	if err == mgo.ErrNotFound {
		return errors.NewNotFoundErrorNEW(errors.ErrTeamNotFound)
	}
	if err != nil {
		Logger.Warn(err.Error())
	}

	return err
}

func (m *Mongore) FindTeamByAlias(alias string) (account.Team, error) {
	team := account.Team{}
	err := m.Teams().Find(bson.M{"alias": alias}).One(&team)

	if err == mgo.ErrNotFound {
		return account.Team{}, errors.NewNotFoundErrorNEW(errors.ErrTeamNotFound)
	}
	if err != nil {
		Logger.Warn(err.Error())
	}

	return team, err
}

func (m *Mongore) DeleteTeamByAlias(alias string) error {
	err := m.Teams().Remove(bson.M{"alias": alias})

	if err == mgo.ErrNotFound {
		return errors.NewNotFoundErrorNEW(errors.ErrTeamNotFound)
	}
	if err != nil {
		Logger.Warn(err.Error())
	}

	return err
}

func (m *Mongore) TeamServices(team account.Team) ([]account.Service, error) {
	services := []account.Service{}
	err := m.Services().Find(bson.M{"team": team.Alias}).All(&services)

	if err != nil {
		Logger.Warn(err.Error())
	}

	return services, err
}

func (m *Mongore) CreateToken(token account.TokenInfo) error {
	key := fmt.Sprintf("%s: %s", token.Type, token.User.Email)
	db.Cache.Set(key, nil, time.Duration(token.Expires)*time.Minute)
	db.HMSET(key, token.Expires, structs.Map(token))

	db.Cache.Set(token.Token, nil, time.Duration(token.Expires))
	db.HMSET(token.Token, token.Expires, structs.Map(token.User))
	return nil
}

func (m *Mongore) DecodeToken(key string, t interface{}) error {
	conn, err := db.Conn()
	if err != nil {
		return err
	}
	defer conn.Close()

	return conn.GetTokenValue(key, t)
}

func (m *Mongore) DeleteToken(key string) error {
	conn, err := db.Conn()
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.DeleteToken(key)
	return err
}

func (m *Mongore) UpsertService(s account.Service) error {
	_, err := m.Services().Upsert(bson.M{"subdomain": s.Subdomain}, s)

	if err != nil {
		Logger.Warn(err.Error())
	}

	return err
}

func (m *Mongore) DeleteService(s account.Service) error {
	err := m.Services().Remove(s)

	if err == mgo.ErrNotFound {
		return errors.NewNotFoundErrorNEW(errors.ErrServiceNotFound)
	}
	if err != nil {
		Logger.Warn(err.Error())
	}

	return err
}

func (m *Mongore) FindServiceBySubdomain(subdomain string) (account.Service, error) {
	var service account.Service
	err := m.Services().Find(bson.M{"subdomain": subdomain}).One(&service)

	if err == mgo.ErrNotFound {
		return account.Service{}, errors.NewNotFoundErrorNEW(errors.ErrServiceNotFound)
	}
	if err != nil {
		Logger.Warn(err.Error())
	}

	return service, err
}

func (m *Mongore) UserServices(user account.User) ([]account.Service, error) {
	var services []account.Service = []account.Service{}

	teams, err := m.UserTeams(user)
	if err != nil {
		Logger.Warn(err.Error())
	}
	var st []string = make([]string, len(teams))
	for i, team := range teams {
		st[i] = team.Alias
	}

	err = m.Services().Find(bson.M{"team": bson.M{"$in": st}}).All(&services)
	return services, err
}

func (m *Mongore) Close() {

}
