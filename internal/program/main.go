package program

import "github.com/kardianos/service"

type Service interface {
	Start(service.Service) error
	Stop() error
}

type program struct {
	service Service
}

func New(service Service) *program {
	return &program{
		service: service,
	}
}

func (p *program) Start(s service.Service) error {
	go p.run(s)
	return nil
}

func (p *program) Stop(s service.Service) error {
	return p.service.Stop()
}

func (p *program) run(s service.Service) {
	p.service.Start(s)
}
