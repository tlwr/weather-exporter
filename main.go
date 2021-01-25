package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	nlogrus "github.com/meatballhat/negroni-logrus"
	"github.com/phyber/negroni-gzip/gzip"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sethvargo/go-signalcontext"
	"github.com/sirupsen/logrus"
	nsecure "github.com/unrolled/secure"
	"github.com/urfave/negroni"
	nprom "github.com/zbindenren/negroni-prometheus"

	"github.com/tlwr/weather-exporter/weather"
)

func main() {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetOutput(os.Stdout)
	logger.SetLevel(logrus.InfoLevel)

	weatherURL := os.Getenv("WEATHER_URL")
	if weatherURL == "" {
		weatherURL = "https://api.met.no/weatherapi/locationforecast/2.0/compact"
	}

	latlonsStr := os.Getenv("WEATHER_LATLONS")
	if latlonsStr == "" {
		latlonsStr = "51.5,0.05"
	}
	latlons := [][]float64{}
	for _, latlonStr := range strings.Split(latlonsStr, " ") {
		split := strings.Split(latlonStr, ",")

		if len(split) != 2 {
			logger.Fatalf(`could not parse "%s" as a lat/lon pair`, latlonStr)
		}

		lat, err := strconv.ParseFloat(split[0], 64)
		if err != nil {
			logger.Fatalf(`could not parse "%s" as a float64`, split[0])
		}

		lon, err := strconv.ParseFloat(split[1], 64)
		if err != nil {
			logger.Fatalf(`could not parse "%s" as a float64`, split[1])
		}

		latlons = append(latlons, []float64{lat, lon})
	}

	var (
		Temperature = promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "weather_temp",
			Help: "The temperature in celsius for lat/lon",
		}, []string{"lat", "lon"})

		WindSpeed = promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "weather_wind_speed",
			Help: "The wind speed in km/ for lat/lon",
		}, []string{"lat", "lon"})

		Precipitation = promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "weather_precipitation",
			Help: "The precipitation in mm for lat/lon",
		}, []string{"lat", "lon"})

		Humidity = promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "weather_humidity",
			Help: "The humidity as a percentage for lat/lon",
		}, []string{"lat", "lon"})
	)

	wc := weather.New(weatherURL)

	go func() {
		fetch := func() {
			for _, latlon := range latlons {
				labels := map[string]string{
					"lat": fmt.Sprintf("%.2f", latlon[0]),
					"lon": fmt.Sprintf("%.2f", latlon[1]),
				}

				data, err := wc.Latest(latlon[0], latlon[1])

				if err != nil {
					logger.Error(err)
					continue
				}

				Temperature.With(labels).Set(data.TemperatureC)
				WindSpeed.With(labels).Set(data.WindSpeedKM)
				Precipitation.With(labels).Set(data.PrecipitationMM)
				Humidity.With(labels).Set(data.HumidityPerc)
			}
		}

		fetch()

		for {
			select {
			case <-time.After(30 * time.Minute):
				fetch()
			}
		}
	}()

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(w, "healthy")
	})

	mux.Handle("/metrics", promhttp.Handler())

	n := negroni.New()
	n.Use(negroni.NewRecovery())
	n.Use(nlogrus.NewMiddlewareFromLogger(logger, "web"))
	n.Use(gzip.Gzip(gzip.DefaultCompression))
	n.Use(negroni.HandlerFunc(nsecure.New().HandlerFuncWithNext))
	n.Use(nprom.NewMiddleware("weather-exporter"))
	n.UseHandler(mux)

	ctx, cancel := signalcontext.On(syscall.SIGTERM)
	defer cancel()

	server := &http.Server{Addr: ":8080", Handler: n}

	go func() {
		server.ListenAndServe()
	}()

	<-ctx.Done()

	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	server.Shutdown(ctx)

	os.Exit(0)
}
