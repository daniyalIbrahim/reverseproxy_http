package reverseproxy
//this reverse proxy supports the http protocol only
import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
	"log"
	"net/http"
	// http2  "golang.org/x/net/http2"
	"net/url"


)


func init(){
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify:true}

}

func main() {

	//HTTP1.1
	demoURL, err := url.Parse("sip://192.168.1.126")

	// HTTP2 requires https
	//demoURL, err := url.Parse("https://192.168.1.1")

	if err != nil{
		log.Fatal(err)

	}
	//proxy := httputil.NewSingleHostReverseProxy(demoURL)

	proxy := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request){
		req.Host = demoURL.Host
		req.URL.Host = demoURL.Host
		req.URL.Scheme = demoURL.Scheme
		req.RequestURI =""

		s ,_ ,_ := net.SplitHostPort(req.RemoteAddr)
		req.Header.Set("X-Forwarded-For",s)

		//http2.ConfigureTransport(http.DefaultTransport.(*http.Transport))



		resp , err := http.DefaultClient.Do(req)
		if err !=nil{
			rw.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(rw,err)
			return
		}


		// These nested loops are required to flush the header
		for key,values := range resp.Header {
			for _,value :=range values{
				rw.Header().Set(key,value)
			}
		}

		// We need to flush more as this is not enough to copy the whole body
		// we need a go routine
		//we make a channel to stop the go routine when the body is copied
		done := make(chan bool)

		go func(){
			for {
				select{
				case <- time.Tick(10*time.Millisecond):
					rw.(http.Flusher).Flush()
					case <- done:
						return
			}
			}
		}()

		trailerKeys :=[]string{}
		for key := range resp.Trailer {
			trailerKeys = append(trailerKeys,key)
		}

		rw.Header().Set("Trailer", strings.Join(trailerKeys, ","))

		rw.WriteHeader(resp.StatusCode)
		fmt.Fprint(rw,resp.StatusCode)
		io.Copy(rw,resp.Body)

		for key,values := range resp.Trailer{
			for _, value :=range values{
				rw.Header().Set(key,value)
			}
		}

		close(done)

	})

	//This is for http
	http.ListenAndServe(":8080",proxy)

	//this is for http2, http2 requires alpn + selected protocol
	//http.ListenAndServeTLS(":8080","cert.pem","key.pem",proxy)
}

/*
Trailer
//Announce Trailers
rw.Header().Set("Trailer","X-Trailer,X-T2")
//Write Header
rw.WriteHeader(http.StatusOK)
//Write Body
rw.Write(body)
//Fill the trailer value
rw.Header().Set("X-trailer","Value")

 */
