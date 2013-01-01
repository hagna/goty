package goty

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

type IRCConn struct {
	Sock        *net.TCPConn
	Read, Write chan string
	Closed      chan bool
}

func Dial(server, nick string) (*IRCConn, error) {
	read := make(chan string, 1000)
	write := make(chan string, 1000)
	closed := make(chan bool)
	con := &IRCConn{nil, read, write, closed}
	err := con.Connect(server, nick)
	return con, err
}

func (con *IRCConn) Connect(server, nick string) error {
	if raddr, err := net.ResolveTCPAddr("tcp", server); err != nil {
		return err
	} else {
		if c, err := net.DialTCP("tcp", nil, raddr); err != nil {
			return err
		} else {
			con.Sock = c
			r := bufio.NewReader(con.Sock)
			w := bufio.NewWriter(con.Sock)

			go func() {
L:
				for {
					select {
					case <-con.Closed:
						fmt.Fprintf(os.Stderr, "goty: read closed\n")
					       	break L 
					default:
						if str, err := r.ReadString(byte('\n')); err != nil {
							fmt.Fprintf(os.Stderr, "goty: read: %s\n", err)
							break L
						} else {
							if strings.HasPrefix(str, "PING") {
								con.Write <- "PONG" + str[4:len(str)-2]
							} else {
								con.Read <- str[0 : len(str)-2]
							}
						}
					}
				}
				fmt.Println("done with send goroutine")
			}()

			go func() {
				for {
					str, ok := <-con.Write
					if ok == false {
						fmt.Fprintf(os.Stderr, "goty: write closed\n")
						break
					}
					if _, err := w.WriteString(str + "\r\n"); err != nil {
						fmt.Fprintf(os.Stderr, "goty: write: %s\n", err)
						break
					}
					w.Flush()
				}
			}()

			con.Write <- "NICK " + nick
			con.Write <- "USER bot * * :..."
		}
	}
	return nil
}

func (con *IRCConn) Close() error {
	con.Closed <- true
	close(con.Write)
	close(con.Read)
	return con.Sock.Close()
}
