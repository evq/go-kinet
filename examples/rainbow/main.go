package main

import (
  "github.com/evq/go-kinet"
  "github.com/lucasb-eyer/go-colorful"
  "image/color"
  "time"
  //"log"
  //"os"
)

func main() {
  // Enable log output
  //log.SetOutput(os.Stdout)

  power_supplies := kinet.DiscoverSupplies()
  ps := power_supplies[0]

  i := 0.0
  for {
    c := colorful.Hcl(i , 0.8, 0.8).Clamped()
    ps.SendColors([]color.Color{c, c, c})
    i = i + 1.0
    if i == 360.0 {
      i = 0.0
    }
    time.Sleep(100 * time.Millisecond)
  }
}
