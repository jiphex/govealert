package mauve

import (
	"net"
	"fmt"
)

type AlertSender interface {
	AddBatchedAlert(alert *Alert)
	SendBatchedAlerts(replace bool) error
}

func LookupMauvesForDomain(domain string) ([]*MauveAlertService, error) {
	cname,addrs,err := net.LookupSRV("mauvealert", "udp", domain)
	if err != nil {
		return nil,fmt.Errorf("Resolution error: %s", err)
	}
	if len(addrs) > 0 {
		ret := make([]*MauveAlertService,len(addrs))
		for i,srv := range addrs {
			ret[i] = &MauveAlertService{
				Host: srv.Target[0:len(srv.Target)-1],
				Port: srv.Port,
			}
		}
		return ret,nil
	} else {
		return nil,fmt.Errorf("Failed to find any Mauve records at %s", cname)
	}
}