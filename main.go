package main

import (
	"encoding/binary"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jessevdk/go-flags"
	"io"
	"log"
	"net"
	"path/filepath"
	"strings"
	"time"
)

type AddQueueNullableString string

func (a AddQueueNullableString) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	inil := "false"
	if a == "" {
		inil = "true"
	}
	uu := struct {
		INil string `xml:"i:nil,attr"`
		Content string `xml:",innerxml"`
	}{
		inil,
		string(a),
	}
	return e.EncodeElement(uu, start)
}

type AddTaskOutputInfo struct {
	DstPath string
	Priority int
	Profile string
}

type AddTaskOutputsConf struct {
	OutputInfo AddTaskOutputInfo
}

type AddQueueItem struct {
	Hash AddQueueNullableString
	Path string
}

type AddTaskTargets struct {
	AddQueueItem AddQueueItem
}

type AddQueueRequest struct {
	AddQueueBat AddQueueNullableString
	DirPath string
	Mode string
	Outputs AddTaskOutputsConf
	RequestId string
	Targets AddTaskTargets
}

func (a AddQueueRequest) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	uu := struct {
		Xmlns string `xml:"xmlns,attr"`
		Xmlnsi string `xml:"xmlns:i,attr"`
		AddQueueBat AddQueueNullableString
		DirPath string
		Mode string
		Outputs AddTaskOutputsConf
		RequestId string
		Targets AddTaskTargets
	}{
		"http://schemas.datacontract.org/2004/07/Amatsukaze.Server",
		"http://www.w3.org/2001/XMLSchema-instance",
		a.AddQueueBat,
		a.DirPath,
		a.Mode,
		a.Outputs,
		a.RequestId,
		a.Targets,
	}
	return e.EncodeElement(uu, start)
}

var opts struct {
	EncodePath  string `short:"e" long:"encode" description:"path to save encoded" required:"true"`
	Input string `short:"i" long:"input" description:"path to input file" required:"true"`
	RemotePath string `short:"r" long:"remote" description:"path on remote device to the folder of input file"`
	Connect string `short:"c" long:"connect" description:"amatsukaze server addr" default:"127.0.0.1:32768"`
	Profile string `short:"p" long:"profile" description:"profile for amatsukaze" required:"true"`
	WolMac string `short:"w" long:"wol" description:"wake on lan mac"`
	WolIface string `short:"I" long:"wol-iface" description:"wake on lan source interface"`
}

func addQueueRequestEncoder(req AddQueueRequest) []byte {
	result := []byte{102, 0} // https://github.com/nekopanda/Amatsukaze/blob/ff26c44a89ad71178605e295868213bdf4a5b958/AmatsukazeServer/Server/ServerInterface.cs#L84
	buf, _ := xml.Marshal(req)
	log.Println(string(buf))
	lenbuf := len(buf)
	lenbufb := make([]byte, 4, 4)
	binary.LittleEndian.PutUint32(lenbufb, uint32(lenbuf))
	lenvec := lenbuf + 4
	lenvecb := make([]byte, 4, 4)
	binary.LittleEndian.PutUint32(lenvecb, uint32(lenvec))
	result = append(result, lenvecb...)
	result = append(result, lenbufb...)
	result = append(result, buf...)
	return result
}

func GetInterfaceIpv4Addr(interfaceName string) (addr string, err error) {
	var (
		ief      *net.Interface
		addrs    []net.Addr
		ipv4Addr net.IP
	)
	if ief, err = net.InterfaceByName(interfaceName); err != nil { // get interface
		return
	}
	if addrs, err = ief.Addrs(); err != nil { // get addresses
		return
	}
	for _, addr := range addrs { // get ipv4 address
		if ipv4Addr = addr.(*net.IPNet).IP.To4(); ipv4Addr != nil {
			break
		}
	}
	if ipv4Addr == nil {
		return "", errors.New(fmt.Sprintf("interface %s don't have an ipv4 address\n", interfaceName))
	}
	return ipv4Addr.String(), nil
}

func wol(mac, iface string) error {
	localAddr, err := GetInterfaceIpv4Addr(iface)
	if err != nil {
		return err
	}
	log.Println("local addr for", iface, ":", localAddr)
	remoteUdpAddr, _ := net.ResolveUDPAddr("udp", "255.255.255.255:9999")
	localUdpAddr, _ := net.ResolveUDPAddr("udp", localAddr + ":0")
	conn, err := net.DialUDP("udp", localUdpAddr, remoteUdpAddr)
	if err != nil {
		return err
	}
	defer func() {
		_ = conn.Close()
	}()

	packetPrefix := make([]byte, 6)
	for i := range packetPrefix {
		packetPrefix[i] = 0xFF
	}

	hw, err := net.ParseMAC(mac)
	if err != nil {
		return err
	}

	sendPacket := make([]byte, 0)
	sendPacket = append(sendPacket, packetPrefix...)
	for i := 0; i < 16; i++ {
		sendPacket = append(sendPacket, hw...)
	}

	_, err = conn.Write(sendPacket)
	return err
}

func tcpMsgDecode(r io.Reader) (uint16, []byte, error) {
	cmdb := []byte{0, 0}
	_, err := io.ReadFull(r, cmdb)
	if err != nil {
		return 0, nil, err
	}
	cmd := binary.LittleEndian.Uint16(cmdb)
	lenb := []byte{0, 0, 0, 0}
	_, err = io.ReadFull(r, lenb)
	if err != nil {
		return 0, nil, err
	}
	leni := binary.LittleEndian.Uint32(lenb)
	content := make([]byte, leni, leni)
	_, err = io.ReadFull(r, content)
	if err != nil {
		return 0, nil, err
	}
	return cmd, content, nil
}

func main() {
	_, err := flags.Parse(&opts)
	if err != nil {
		log.Fatalln("failed on parse flags; ", err)
	}

	inputPath := opts.Input
	if opts.RemotePath != "" {
		remotePath := opts.RemotePath
		if !strings.HasSuffix(remotePath, "\\") {
			remotePath += "\\"
		}
		inputPath = remotePath + filepath.Base(inputPath)
	}

	requestId := uuid.New().String()
	addRequest := AddQueueRequest{
		"",
		"",
		"AutoBatch",
		AddTaskOutputsConf{
			AddTaskOutputInfo{
				opts.EncodePath,
				3,
				opts.Profile,
			},
		},
		requestId,
		AddTaskTargets{
			AddQueueItem{
				"",
				inputPath,
			},
		},
	}

	if opts.WolMac != "" && opts.WolIface != "" {
		if err := wol(opts.WolMac, opts.WolIface); err != nil {
			log.Fatalln("failed on wol; ", err)
		}
		time.Sleep(10*time.Second)
	}

	d := net.Dialer{Timeout: 5 * time.Second}
	conn, err := d.Dial("tcp", opts.Connect)
	if err != nil {
		log.Fatalln("failed on connect to amatsukaze server; ", err)
	}
	defer conn.Close()
	cmdToWrite := addQueueRequestEncoder(addRequest)
	_, err = conn.Write(cmdToWrite)
	if err != nil {
		log.Fatalln("failed on send add request; ", err)
	}
	for i := 0; i < 10000; i++ {
		cmd, content, err := tcpMsgDecode(conn)
		if err != nil {
			log.Fatalln("failed on reading response; ", err)
		}
		log.Println(string(content[4:]))
		if cmd == 210 { // https://github.com/nekopanda/Amatsukaze/blob/ff26c44a89ad71178605e295868213bdf4a5b958/AmatsukazeServer/Server/ServerInterface.cs#L108
			break
		}
	}
}
