package wecom

import (
	"encoding/xml"
)

// Incoming XML envelope from WeChat Work callback.
type xmlEncryptedMsg struct {
	XMLName    xml.Name `xml:"xml"`
	ToUserName string   `xml:"ToUserName"`
	AgentID    string   `xml:"AgentID"`
	Encrypt    string   `xml:"Encrypt"`
}

// Decrypted message body.
type xmlMessage struct {
	XMLName      xml.Name `xml:"xml"`
	ToUserName   string   `xml:"ToUserName"`
	FromUserName string   `xml:"FromUserName"`
	CreateTime   int64    `xml:"CreateTime"`
	MsgType      string   `xml:"MsgType"`
	Content      string   `xml:"Content"`
	PicUrl       string   `xml:"PicUrl"`
	MediaId      string   `xml:"MediaId"`
	Format       string   `xml:"Format"` // voice format: amr, speex, etc.
	MsgId        int64    `xml:"MsgId"`
	AgentID      int64    `xml:"AgentID"`
}
