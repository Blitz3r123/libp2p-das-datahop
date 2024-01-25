package main

import (
	"encoding/json"
	"log"
	"time"
)
type EventCode int
const (
    HeaderSent EventCode = iota
    HeaderReceived
    SamplingFinished
)
    
    

type LogEvent struct {
	Timestamp string `json:"timestamp"`
	EventType EventCode    `json:"eventType"`
	BlockId   int    `json:"blockId"`
}

func formatJSONLogEvent(eventType EventCode, blockId int) string {
	// Custom log entry struct
	logEntry := LogEvent{
		Timestamp: time.Now().Format(time.RFC3339Nano),
		EventType: eventType,
		BlockId:   blockId,
	}

	// Marshal log entry to JSON
	jsonData, err := json.Marshal(logEntry)
	if err != nil {
		log.Println("Error marshaling JSON:", err)
		return ""
	}

	return string(jsonData)
}
