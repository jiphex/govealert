package mauve

import (
	"fmt"
	"sync"
	"log"
	"net"
	
	"code.google.com/p/goprotobuf/proto"
)
type ProtobufClient struct {
	Hosts []*MauveAlertService
	Source string
	
	// Some internal fields
	batchedAlerts []*Alert
}

func CreateProtobufClient(source string, domain string) (*ProtobufClient,error) {
	pbc := &ProtobufClient{}
	pbc.Source = source
	pbc.batchedAlerts = make([]*Alert,0)
	ph,err := LookupMauvesForDomain(domain)
	if err != nil {
		return nil,fmt.Errorf("Failed to lookup Mauve for %s: %s", domain, err)
	}
	pbc.Hosts = ph
	return pbc,nil
}

func (pbc *ProtobufClient) AddBatchedAlert(alert *Alert) {
	pbc.batchedAlerts = append(pbc.batchedAlerts,alert)
}

func (pbc *ProtobufClient) SendBatchedAlerts(replace bool) error {
	wg := &sync.WaitGroup{}
	up := CreateUpdate(pbc.Source, replace, pbc.batchedAlerts...)
	wg.Add(len(pbc.Hosts))
	for _,srv := range pbc.Hosts {
		go func() {
			defer wg.Done()
			// This connects to Mauve over UDP and then waits on it's channel,
			// any Alert that gets written to the channel will get wrapped in
			// an AlertUpdate and then sent to the Mauve server
			mauveIP,err := net.ResolveIPAddr("ip",srv.Host)
			addr := &net.UDPAddr{mauveIP.IP,int(srv.Port),mauveIP.Zone}
			if err != nil {
				log.Fatalf("Cannot resolve mauvealert server: %s", addr)
			}
			conn, err := net.DialUDP("udp", nil, addr)
			if err != nil {
				log.Fatalf("Failed to connect to mauve: %s", addr)
			}
			defer conn.Close() // Just make sure that the connection gets flushed
			//log.Printf("dialing...")
			mu, err := proto.Marshal(up)
			if err != nil {
				log.Fatalf("Failed to marshal an alertUpdate: %s", err)
			}
			//log.Printf("Sent: %s", up.String())
			if bytes, err := conn.Write(mu); err != nil {
				log.Fatalf("Failed to send message: %s", err)
			} else {
				log.Printf("Sent %d bytes to %s:%d", bytes, srv.Host, srv.Port)
			}
		}()
	}
	wg.Wait()
	return nil
}
