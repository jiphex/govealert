package mauve

import (
	"fmt"
	"log"
	
	
	"code.google.com/p/goprotobuf/proto"
	 mqtt "git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git"
)

type MQTTClient struct {
	Broker string
	BaseTopic string
	Source string
	
	// non-exported fields
	batchedAlerts []*Alert
}

func CreateMQTTClient(source string, broker string, baseTopic string) (*MQTTClient,error) {
	mqc := &MQTTClient{
		Broker: broker,
		BaseTopic: baseTopic,
		Source: source,
	}
	mqc.batchedAlerts = make([]*Alert, 0)
	return mqc,nil
}

func (mqc *MQTTClient) AddBatchedAlert(alert *Alert) {
	mqc.batchedAlerts = append(mqc.batchedAlerts,alert)
}

func alertTopic(al *Alert, source string) string {
    return fmt.Sprintf("%s/%s/%s", source, *al.Subject, *al.Id)
}

func (mqc *MQTTClient) SendBatchedAlerts(replace bool) error {
	var err error
	mqttOpts := mqtt.NewClientOptions().AddBroker(mqc.Broker).SetClientId(RandomID()).SetCleanSession(true).SetOnConnectionLost(func(client *mqtt.MqttClient, reason error){
		err = reason
	})
	client := mqtt.NewClient(mqttOpts)
	if _, err = client.Start(); err != nil {
		return err
	} else {
	    log.Printf("Connected to Broker")
	}
	if replace {
		log.Printf("Not possible to use replace with MQTT")
	}
	// There's no real notion of Updates or Replace in MQTT
	for _,al := range mqc.batchedAlerts {
	    var pkt []byte
	    var err error
	    pkt, err = proto.Marshal(al)
	    if err != nil {
	        log.Fatalf("Marshalling fail: %v", err)
	    }
	    // send the packet
	    log.Printf("Sending MQTT transport packet: %s", al)
	    fullTopic := fmt.Sprintf("%s/%s", mqc.BaseTopic, alertTopic(al, mqc.Source))
	    mqttreceipt := client.Publish(mqtt.QOS_ONE, fullTopic, pkt)
	    <-mqttreceipt
	    log.Printf("Sent MQTT transport packet: %s", al)
	}
	return err
}