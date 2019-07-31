package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	consul "github.com/hashicorp/consul/api"
)

const (
	_VERSION = "0.8.3"
	_PREFIX  = "DNSMASQ/"
)

var (
	opts = Options{}
)

type Options struct {
	Version bool
}

type Lease struct {
	Ip         string `json:"ip"`
	Mac        string `json:"mac"`
	Hostname   string `json:"host"`
	Domain     string `json:"domain"`
	If         string `json:"interface"`
	Expire     string `json:"expire"`
	RemainTime string `json:"remaining_time"`
	Cmd        string `json:"cmd"`
	ClientID   string `json:"client_id"`
	Timestamp  int64  `json:"timestamp"`
	Time       string `json:"time"`
}

type Consul struct {
	cli *consul.Client
	kv  *consul.KV
}

func init() {
	flag.BoolVar(&opts.Version, "v", false, "Version")
}

func newClient() *Consul {
	client, err := consul.NewClient(consul.DefaultConfig())
	if err != nil {
		panic(err)
	}

	cl := &Consul{}
	cl.cli = client
	cl.kv = client.KV()

	return cl
}

func (cl *Consul) getKeys() ([]Lease, error) {
	var retval []Lease

	keys, _, err := cl.kv.Keys(_PREFIX, "", nil)
	if err != nil {
		return nil, err
	}

	for _, v := range keys {
		lease, err := cl.getKey(v)
		if err != nil {
			return nil, err
		}

		retval = append(retval, *lease)
	}

	return retval, nil
}

func (cl *Consul) getKey(key string) (*Lease, error) {
	retval := &Lease{}

	kv, _, err := cl.kv.Get(key, nil)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(kv.Value, retval)
	if err != nil {
		return nil, err
	}

	//	fmt.Printf("%+v\n", retval)

	return retval, nil
}

func (cl *Consul) putKey(lease *Lease) error {
	val, err := json.Marshal(lease)
	if err != nil {
		return err
	}

	p := &consul.KVPair{Key: _PREFIX + lease.Ip, Value: val}

	_, err = cl.kv.Put(p, nil)
	if err != nil {
		return err
	}

	return nil
}

func (cl *Consul) delKey(key string) error {
	_, err := cl.kv.Delete(key, nil)
	if err != nil {
		return err
	}

	return nil
}

func fillLease(lease *Lease) {
	now := time.Now()
	lease.Domain = os.Getenv("DNSMASQ_DOMAIN")
	lease.RemainTime = os.Getenv("DNSMASQ_TIME_REMAINING")
	lease.If = os.Getenv("DNSMASQ_INTERFACE")
	lease.ClientID = os.Getenv("DNSMASQ_CLIENT_ID")
	if lease.ClientID == "" {
		lease.ClientID = "*"
	}
	lease.Timestamp = now.Unix()
	lease.Expire = os.Getenv("DNSMASQ_LEASE_EXPIRES")
	if lease.Expire == "" {
		lease.Expire = "*"
	}
	lease.Time = now.Format("2006-01-02T15:04:05.000Z")

	//	fmt.Println(lease)
}

func main() {
	var lease = &Lease{}

	flag.Parse()

	if opts.Version {
		fmt.Printf("%s: %s\n", os.Args[0], _VERSION)
		os.Exit(0)
	}

	switch len(os.Args) {
	case 2:
		lease.Cmd = os.Args[1]
	case 5:
		lease.Cmd = os.Args[1]
		lease.Mac = os.Args[2]
		lease.Ip = os.Args[3]
		lease.Hostname = os.Args[4]
	}

	cl := newClient()

	switch lease.Cmd {
	case "add":
	case "old":
		// old 28:6c:07:85:ed:ea 192.168.1.222 xiaomi_gateway
		fillLease(lease)
		err := cl.putKey(lease)
		if err != nil {
			panic(err)
		}
	case "init":
		// 1564497486 04:b1:67:34:0c:d2 192.168.1.29 xiaomi_a1 01:04:b1:67:34:0c:d2
		var dnsmasq string
		list, err := cl.getKeys()
		if err != nil {
			panic(err)
		}
		for _, v := range list {
			dnsmasq += v.Expire + " " + v.Mac + " " + v.Ip + " " + v.Hostname + " " + v.ClientID + "\n"
		}
		fmt.Println(dnsmasq)
		os.Exit(0)
	case "del":
		fillLease(lease)
		err := cl.delKey(_PREFIX + lease.Ip)
		if err != nil {
			panic(err)
		}
	case "arp-add":
	case "arp-del":
	case "tftp":
	default:
	}
}
