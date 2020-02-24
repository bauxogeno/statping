// Statping
// Copyright (C) 2018.  Hunter Long and the project contributors
// Written by Hunter Long <info@socialeck.com> and the project contributors
//
// https://github.com/hunterlong/statping
//
// The licenses for most software and other practical works are designed
// to take away your freedom to share and change the works.  By contrast,
// the GNU General Public License is intended to guarantee your freedom to
// share and change all versions of a program--to make sure it remains free
// software for all its users.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package core

import (
	"fmt"
	"github.com/hunterlong/statping/core/notifier"
	"github.com/hunterlong/statping/database"
	"github.com/hunterlong/statping/types"
	"github.com/hunterlong/statping/utils"
	"sort"
	"strconv"
	"time"
)

type Service struct {
	*types.Service
}

// Select will return the *types.Service struct for Service
func (s *Service) Select() *types.Service {
	return s.Service
}

// ReturnService will convert *types.Service to *core.Service
func ReturnService(s *types.Service) *Service {
	return &Service{s}
}

func Services() []types.ServiceInterface {
	return CoreApp.Services
}

// SelectService returns a *core.Service from in memory
func SelectService(id int64) *Service {
	for _, s := range Services() {
		if s.Select().Id == id {
			service := s.(*Service)
			service = service.UpdateStats()
			return service
		}
	}
	return nil
}

func (s *Service) GetFailures(count int) []*Failure {
	var fails []*Failure
	db := Database(&types.Failure{}).Where("service = ?", s.Id)
	db.Limit(count).Find(&fails)
	return fails
}

func (s *Service) UpdateStats() *Service {
	s.Online24Hours = s.OnlineDaysPercent(1)
	s.Online7Days = s.OnlineDaysPercent(7)
	s.AvgResponse = s.AvgTime()
	s.FailuresLast24Hours = s.FailuresDaysAgo(1)
	s.LastFailure = s.lastFailure()
	return s
}

func SelectServices(auth bool) []*Service {
	var validServices []*Service
	for _, sr := range CoreApp.Services {
		s := sr.(*Service)
		if !s.Public.Bool {
			if auth {
				validServices = append(validServices, s)
			}
		} else {
			validServices = append(validServices, s)
		}
	}
	return validServices
}

// SelectServiceLink returns a *core.Service from the service permalink
func SelectServiceLink(permalink string) *Service {
	for _, s := range Services() {
		if s.Select().Permalink.String == permalink {
			return s.(*Service)
		}
	}
	return nil
}

// CheckinProcess runs the checkin routine for each checkin attached to service
func (s *Service) CheckinProcess() {
	checkins := s.AllCheckins()
	for _, c := range checkins {
		c.Start()
		go c.Routine()
	}
}

// AllCheckins will return a slice of AllCheckins for a Service
func (s *Service) AllCheckins() []*Checkin {
	var checkin []*Checkin
	Database(&Checkin{}).Where("service = ?", s.Id).Find(&checkin)
	return checkin
}

// SelectAllServices returns a slice of *core.Service to be store on []*core.Services
// should only be called once on startup.
func (c *Core) SelectAllServices(start bool) ([]*Service, error) {
	var services []*Service
	db := Database(&Service{}).Find(&services).Order("order_id desc")
	if db.Error() != nil {
		log.Errorln(fmt.Sprintf("service error: %v", db.Error()))
		return nil, db.Error()
	}
	CoreApp.Services = nil
	for _, service := range services {
		if start {
			service.Start()
			service.CheckinProcess()
		}
		fails := service.GetFailures(limitedFailures)
		for _, f := range fails {
			service.Failures = append(service.Failures, f)
		}
		checkins := service.AllCheckins()
		for _, c := range checkins {
			c.Failures = c.GetFailures(limitedFailures)
			c.Hits = c.LimitedHits(limitedHits)
			service.Checkins = append(service.Checkins, c)
		}
		// collect initial service stats
		service = service.UpdateStats()
		CoreApp.Services = append(CoreApp.Services, service)
	}
	reorderServices()
	return services, db.Error()
}

// reorderServices will sort the services based on 'order_id'
func reorderServices() {
	sort.Sort(ServiceOrder(CoreApp.Services))
}

// AvgTime will return the average amount of time for a service to response back successfully
func (s *Service) AvgTime() float64 {
	total, _ := s.TotalHits()
	if total == 0 {
		return 0
	}
	avg := s.Sum() / float64(total) * 100
	f, _ := strconv.ParseFloat(fmt.Sprintf("%0.0f", avg*10), 32)
	return f
}

// OnlineDaysPercent returns the service's uptime percent within last 24 hours
func (s *Service) OnlineDaysPercent(days int) float32 {
	ago := time.Now().UTC().Add((-24 * time.Duration(days)) * time.Hour)
	return s.OnlineSince(ago)
}

// OnlineSince accepts a time since parameter to return the percent of a service's uptime.
func (s *Service) OnlineSince(ago time.Time) float32 {
	failed, _ := s.TotalFailuresSince(ago)
	if failed == 0 {
		s.Online24Hours = 100.00
		return s.Online24Hours
	}
	total, _ := s.TotalHitsSince(ago)
	if total == 0 {
		s.Online24Hours = 0
		return s.Online24Hours
	}
	avg := float64(failed) / float64(total) * 100
	avg = 100 - avg
	if avg < 0 {
		avg = 0
	}
	amount, _ := strconv.ParseFloat(fmt.Sprintf("%0.2f", avg), 10)
	s.Online24Hours = float32(amount)
	return s.Online24Hours
}

// lastFailure returns the last Failure a service had
func (s *Service) lastFailure() types.FailureInterface {
	limited := s.GetFailures(1)
	if len(limited) == 0 {
		return nil
	}
	last := limited[len(limited)-1]
	return last
}

// DowntimeText will return the amount of downtime for a service based on the duration
//		service.DowntimeText()
//		// Service has been offline for 15 minutes
func (s *Service) DowntimeText() string {
	return fmt.Sprintf("%v has been offline for %v", s.Name, utils.DurationReadable(s.Downtime()))
}

// Dbtimestamp will return a SQL query for grouping by date
func Dbtimestamp(group string, column string) string {
	seconds := 3600
	switch group {
	case "minute":
		seconds = 60
	case "hour":
		seconds = 3600
	case "day":
		seconds = 86400
	case "week":
		seconds = 604800
	case "month":
		seconds = 2592000
	case "year":
		seconds = 31557600
	default:
		seconds = 60
	}
	switch CoreApp.Config.DbConn {
	case "mysql":
		return fmt.Sprintf("CONCAT(date_format(created_at, '%%Y-%%m-%%d %%H:00:00')) AS timeframe, AVG(%v) AS value", column)
	case "postgres":
		return fmt.Sprintf("date_trunc('%v', created_at) AS timeframe, AVG(%v) AS value", group, column)
	default:
		return fmt.Sprintf("datetime((strftime('%%s', created_at) / %v) * %v, 'unixepoch') AS timeframe, AVG(%v) as value", seconds, seconds, column)
	}
}

// Downtime returns the amount of time of a offline service
func (s *Service) Downtime() time.Duration {
	hits, _ := s.Hits()
	fail := s.lastFailure()
	if fail == nil {
		return time.Duration(0)
	}
	if len(hits) == 0 {
		return time.Now().UTC().Sub(fail.Select().CreatedAt.UTC())
	}
	since := fail.Select().CreatedAt.UTC().Sub(hits[0].CreatedAt.UTC())
	return since
}

// GraphData will return all hits or failures
func GraphData(q *database.GroupQuery, dbType interface{}, by database.By) []*database.TimeValue {
	dbQuery, err := q.Database().GroupQuery(q, by).ToTimeValue(dbType)

	if err != nil {
		log.Error(err)
		return nil
	}
	if q.FillEmpty {
		return dbQuery.FillMissing(q.Start, q.End)
	}
	return dbQuery.ToValues()
}

// index returns a services index int for updating the []*core.Services slice
func (s *Service) index() int {
	for k, service := range CoreApp.Services {
		if s.Id == service.(*Service).Id {
			return k
		}
	}
	return 0
}

// updateService will update a service in the []*core.Services slice
func updateService(s *Service) {
	CoreApp.Services[s.index()] = s
}

// Delete will remove a service from the database, it will also end the service checking go routine
func (s *Service) Delete() error {
	i := s.index()
	err := Database(&Service{}).Delete(s)
	if err.Error() != nil {
		log.Errorln(fmt.Sprintf("Failed to delete service %v. %v", s.Name, err.Error()))
		return err.Error()
	}
	s.Close()
	slice := CoreApp.Services
	CoreApp.Services = append(slice[:i], slice[i+1:]...)
	reorderServices()
	notifier.OnDeletedService(s.Service)
	return err.Error()
}

// Update will update a service in the database, the service's checking routine can be restarted by passing true
func (s *Service) Update(restart bool) error {
	err := Database(&Service{}).Update(&s)
	if err.Error() != nil {
		log.Errorln(fmt.Sprintf("Failed to update service %v. %v", s.Name, err))
		return err.Error()
	}
	// clear the notification queue for a service
	if !s.AllowNotifications.Bool {
		for _, n := range CoreApp.Notifications {
			notif := n.(notifier.Notifier).Select()
			notif.ResetUniqueQueue(fmt.Sprintf("service_%v", s.Id))
		}
	}
	if restart {
		s.Close()
		s.Start()
		s.SleepDuration = time.Duration(s.Interval) * time.Second
		go s.CheckQueue(true)
	}
	reorderServices()
	updateService(s)
	notifier.OnUpdatedService(s.Service)
	return err.Error()
}

// Create will create a service and insert it into the database
func (s *Service) Create(check bool) (int64, error) {
	s.CreatedAt = time.Now().UTC()
	db := Database(&Service{}).Create(s)
	if db.Error() != nil {
		log.Errorln(fmt.Sprintf("Failed to create service %v #%v: %v", s.Name, s.Id, db.Error()))
		return 0, db.Error()
	}
	s.Start()
	go s.CheckQueue(check)
	CoreApp.Services = append(CoreApp.Services, s)
	reorderServices()
	notifier.OnNewService(s.Service)
	return s.Id, nil
}

// Messages returns all Messages for a Service
func (s *Service) Messages() []*Message {
	messages := SelectServiceMessages(s.Id)
	return messages
}

// ActiveMessages returns all service messages that are available based on the current time
func (s *Service) ActiveMessages() []*Message {
	var messages []*Message
	msgs := SelectServiceMessages(s.Id)
	for _, m := range msgs {
		if m.StartOn.UTC().After(time.Now().UTC()) {
			messages = append(messages, m)
		}
	}
	return messages
}

// CountOnline returns the amount of services online
func (c *Core) CountOnline() int {
	amount := 0
	for _, s := range CoreApp.Services {
		if s.Select().Online {
			amount++
		}
	}
	return amount
}
