//
// Simple app to query DNS for some records based on a config file
// and then verify they match.
//

package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strings"

	"github.com/miekg/dns"
	"github.com/urfave/cli"
	"gopkg.in/yaml.v2"
)

var changedata = map[string]map[string]string{}

func main() {

	var configfile string
	var domain string
	var nameserver string

	app := cli.NewApp()

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "config, c",
			Usage:       "Load configuration from `FILE`",
			Destination: &configfile,
		},
		cli.StringFlag{
			Name:        "domain, d",
			Usage:       "Domain to query against",
			Destination: &domain,
		},
		cli.StringFlag{
			Name:        "nameserver, n",
			Usage:       "nameserver to query against",
			Destination: &nameserver,
		},
	}

	app.Action = func(c *cli.Context) error {

		if configfile == "" {
			log.Fatal("please specify a conifg file")
		}
		if domain == "" {
			log.Fatal("please specify a domain")
		}
		if nameserver == "" {
			log.Fatal("please specify a nameserver ip")
		} else {
			nameserver = ipfromhostname(nameserver)
		}

		data, err := ioutil.ReadFile(configfile)
		if err != nil {
			log.Fatalln(err)
		}

		results := make(map[interface{}]map[interface{}]string)
		err = yaml.Unmarshal(data, &results)
		if err != nil {
			log.Fatalln(err)
		}

		for k, va := range results {
			var key string
			key = k.(string)
			query(key, va, domain, nameserver)
		}

		//fmt.Println(changedata)
		y, err := yaml.Marshal(changedata)
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Println(string(y))
		d2 := []byte(y)

		err = ioutil.WriteFile("./changes.yaml", d2, 0644)
		check(err)

		return nil
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}

}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func rrtypecheck(ans dns.RR, target string, rtype string) error {
	rrtype := strings.Fields(ans.Header().String())
	if rrtype[3] != rtype {
		fmt.Printf("[ERR] %s is not a type %s record it is a type %s record\n", target, rtype, rrtype[3])
		return errors.New("bad record type")
	}
	return nil
}

func ipfromhostname(nameserver string) (addr string) {
	ip := net.ParseIP(nameserver)
	//fmt.Printf("ip is: %s", ip)
	if ip == nil {
		target := dns.Fqdn(nameserver)
		server := "8.8.8.8"
		c := dns.Client{}
		msg := dns.Msg{}
		msg.SetQuestion(target, dns.TypeA)
		r, _, err := c.Exchange(&msg, server+":53")
		if err != nil {
			fmt.Printf("[ERR] %s query timed out\n", nameserver)
			return
		}
		if len(r.Answer) == 0 {
			log.Fatal("No results")
		}
		for _, ans := range r.Answer {
			Arecord := ans.(*dns.A)
			addr = Arecord.A.String()
		}

		//fmt.Printf("addr: %s\n", addr)
	} else {
		addr = nameserver
	}
	return
}

func query(name string, record map[interface{}]string, domain string, nameserver string) {
	target := dns.Fqdn(name + "." + domain)
	server := nameserver
	rtype := record["type"]
	rvalue := record["value"]
	var value string

	c := dns.Client{}
	msg := dns.Msg{}
	if rtype == "A" {
		msg.SetQuestion(target, dns.TypeA)
	}
	if rtype == "AAAA" {
		msg.SetQuestion(target, dns.TypeAAAA)
	}
	if rtype == "PTR" {
		msg.SetQuestion(target, dns.TypePTR)
	}
	if rtype == "CNAME" {
		msg.SetQuestion(target, dns.TypeCNAME)
	}
	if rtype == "TXT" {
		msg.SetQuestion(target, dns.TypeTXT)
	}

	r, _, err := c.Exchange(&msg, server+":53")
	if err != nil {
		fmt.Printf("[ERR] %s.%s query timed out\n", name, domain)
		return
	}
	if len(r.Answer) == 0 {
		log.Fatal("No results")
	}
	for _, ans := range r.Answer {
		if rtype == "A" {
			err := rrtypecheck(ans, target, rtype)
			if err != nil {
				return
			}
			Arecord := ans.(*dns.A)
			value = Arecord.A.String()
			if rvalue != value {
				fmt.Printf("[CHG] %s.%s records did not match\n", name, domain)
				changedata[name] = map[string]string{}
				changedata[name]["add"] = rvalue
				changedata[name]["delete"] = value
			} else {
				fmt.Printf("[OK] %s.%s records matched\n", name, domain)
			}

		}
		if rtype == "AAAA" {
			err := rrtypecheck(ans, target, rtype)
			if err != nil {
				return
			}
			Arecord := ans.(*dns.AAAA)
			value = Arecord.AAAA.String()
			if rvalue != value {
				fmt.Printf("[CHG] %s.%s records did not match\n", name, domain)
				changedata[name] = map[string]string{}
				changedata[name]["add"] = rvalue
				changedata[name]["delete"] = value
			} else {
				fmt.Printf("[OK] %s.%s records matched\n", name, domain)
			}
		}
		if rtype == "PTR" {
			err := rrtypecheck(ans, target, rtype)
			if err != nil {
				return
			}
			Arecord := ans.(*dns.PTR)
			value = Arecord.Ptr
			if rvalue != value {
				fmt.Printf("[CHG] %s.%s records did not match\n", name, domain)
				changedata[name] = map[string]string{}
				changedata[name]["add"] = rvalue
				changedata[name]["delete"] = value
			} else {
				fmt.Printf("[OK] %s.%s records matched\n", name, domain)
			}
		}
		if rtype == "CNAME" {
			err := rrtypecheck(ans, target, rtype)
			if err != nil {
				return
			}
			Arecord := ans.(*dns.CNAME)
			value = Arecord.Target
			if rvalue != value {
				fmt.Printf("[CHG] %s.%s records did not match\n", name, domain)
				changedata[name] = map[string]string{}
				changedata[name]["add"] = rvalue
				changedata[name]["delete"] = value
			} else {
				fmt.Printf("[OK] %s.%s records matched\n", name, domain)
			}
		}
		if rtype == "TXT" {
			err := rrtypecheck(ans, target, rtype)
			if err != nil {
				return
			}
			Arecord := ans.(*dns.TXT)
			value = Arecord.Txt[0]
			if rvalue != value {
				fmt.Printf("[CHG] %s.%s records did not match\n", name, domain)
				changedata[name] = map[string]string{}
				changedata[name]["add"] = rvalue
				changedata[name]["delete"] = value
			} else {
				fmt.Printf("[OK] %s.%s records matched\n", name, domain)
			}
		}
	}
}
