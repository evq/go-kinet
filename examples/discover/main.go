package main

import (
  "github.com/evq/go-kinet"
  "fmt"
  //"log"
  //"os"
  "encoding/json"
)

func main() {
  // Enable log output
  //log.SetOutput(os.Stdout)

  power_supplies := kinet.Discover()
  for i := range power_supplies {
    ps, _ := json.Marshal(power_supplies[i])
    fmt.Println(string(ps))
  }
}
