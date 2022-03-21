package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"syscall/js"

	"cloud.google.com/go/storage"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
)

func main() {
	fmt.Println("============================================")
	fmt.Println("init")
	fmt.Println("============================================")
	fmt.Println()
	js.Global().Set("GetGCSFile", GetGCSFile())
	js.Global().Set("GetGCSFileStream", GetGCSFileStream())
	<-make(chan bool)
}

func GetGCSFile() js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		bucketName := args[0].String()
		fileName := args[1].String()
		token := args[2].String()
		handler := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			resolve := args[0]
			reject := args[1]
			go func() {

				ctx := context.Background()
				ts := oauth2.StaticTokenSource(&oauth2.Token{
					AccessToken: token,
					//Expiry:      time.Now().Add(time.Duration(30 * time.Second)),
				})
				client, err := storage.NewClient(ctx, option.WithTokenSource(ts))
				if err != nil {
					fmt.Printf("Error: %v", err)
					errorConstructor := js.Global().Get("Error")
					errorObject := errorConstructor.New(err.Error())
					reject.Invoke(errorObject)
					return
				}
				bkt := client.Bucket(bucketName)
				obj := bkt.Object(fileName)
				res, err := obj.NewReader(ctx)
				if err != nil {
					fmt.Printf("Error: %v", err)
					errorConstructor := js.Global().Get("Error")
					errorObject := errorConstructor.New(err.Error())
					reject.Invoke(errorObject)
					return
				}

				data, err := ioutil.ReadAll(res)
				if err != nil {
					errorConstructor := js.Global().Get("Error")
					errorObject := errorConstructor.New(err.Error())
					reject.Invoke(errorObject)
					return
				}

				arrayConstructor := js.Global().Get("Uint8Array")
				dataJS := arrayConstructor.New(len(data))
				js.CopyBytesToJS(dataJS, data)

				responseConstructor := js.Global().Get("Response")
				response := responseConstructor.New(dataJS)

				resolve.Invoke(response)
			}()
			return nil
		})
		promiseConstructor := js.Global().Get("Promise")
		return promiseConstructor.New(handler)
	})
}

func GetGCSFileStream() js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		bucketName := args[0].String()
		fileName := args[1].String()
		token := args[2].String()
		handler := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			resolve := args[0]
			reject := args[1]
			go func() {

				ctx := context.Background()
				ts := oauth2.StaticTokenSource(&oauth2.Token{
					AccessToken: token,
					//Expiry:      time.Now().Add(time.Duration(30 * time.Second)),
				})
				client, err := storage.NewClient(ctx, option.WithTokenSource(ts))
				if err != nil {
					fmt.Printf("Error: %v", err)
					errorConstructor := js.Global().Get("Error")
					errorObject := errorConstructor.New(err.Error())
					reject.Invoke(errorObject)
					return
				}
				bkt := client.Bucket(bucketName)
				obj := bkt.Object(fileName)
				r, err := obj.NewReader(ctx)
				if err != nil {
					fmt.Printf("Error: %v", err)
					errorConstructor := js.Global().Get("Error")
					errorObject := errorConstructor.New(err.Error())
					reject.Invoke(errorObject)
					return
				}

				underlyingSource := map[string]interface{}{
					"start": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
						controller := args[0]

						go func() {
							defer r.Close()
							for {
								// Read up to 16KB at a time
								buf := make([]byte, 2*16384)
								n, err := r.Read(buf)
								if err != nil && err != io.EOF {
									errorConstructor := js.Global().Get("Error")
									errorObject := errorConstructor.New(err.Error())
									controller.Call("error", errorObject)
									return
								}
								if n > 0 {

									arrayConstructor := js.Global().Get("Uint8Array")
									dataJS := arrayConstructor.New(n)
									js.CopyBytesToJS(dataJS, buf[0:n])
									controller.Call("enqueue", dataJS)
								}
								if err == io.EOF {
									controller.Call("close")
									return
								}
							}
						}()

						return nil
					}),

					"cancel": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
						r.Close()
						return nil
					}),
				}

				readableStreamConstructor := js.Global().Get("ReadableStream")
				readableStream := readableStreamConstructor.New(underlyingSource)
				responseInitObj := map[string]interface{}{
					"status":     http.StatusOK,
					"statusText": http.StatusText(http.StatusOK),
				}
				responseConstructor := js.Global().Get("Response")
				response := responseConstructor.New(readableStream, responseInitObj)
				resolve.Invoke(response)
			}()
			return nil
		})

		promiseConstructor := js.Global().Get("Promise")
		return promiseConstructor.New(handler)
	})
}
