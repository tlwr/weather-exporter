package weather_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	. "github.com/onsi/gomega/gstruct"

	"github.com/tlwr/weather-exporter/weather"
)

func TestSuite(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "weather")
}

var _ = Describe("WeatherClient", func() {
	var (
		wc weather.WeatherClient

		s *ghttp.Server
	)

	BeforeEach(func() {
		s = ghttp.NewServer()
		wc = weather.New(s.URL())
	})

	AfterEach(func() {
		s.Close()
	})

	It("gets the latest weather", func() {
		s.AppendHandlers(
			ghttp.RespondWith(200, `
{
  "type": "Feature",
  "geometry": {
    "type": "Point",
    "coordinates": [
      0,
      51.5,
      4
    ]
  },
  "properties": {
    "timeseries": [
      {
        "time": "2021-01-25T13:00:00Z",
        "data": {
          "instant": {
            "details": {
              "air_pressure_at_sea_level": 1010.5,
              "air_temperature": 3.7,
              "cloud_area_fraction": 2.3,
              "relative_humidity": 76.2,
              "wind_from_direction": 275.3,
              "wind_speed": 4.3
            }
          },
          "next_1_hours": {
            "summary": {
              "symbol_code": "clearsky_day"
            },
            "details": {
              "precipitation_amount": 0
            }
          }
        }
      }
    ]
  }
}
`),
		)
		data, err := wc.Latest(51.5, 0.05)

		Expect(err).NotTo(HaveOccurred())

		Expect(*data).To(MatchAllFields(Fields{
			"TemperatureC":    Equal(3.7),
			"WindSpeedKM":     Equal(4.3),
			"PrecipitationMM": Equal(0.0),
			"HumidityPerc":    Equal(76.2),
		}))
	})

	Context("when not HTTP 200", func() {
		It("returns an error", func() {
			s.AppendHandlers(
				ghttp.RespondWith(403, "nei takk"),
			)

			data, err := wc.Latest(51.5, 0.05)

			Expect(data).To(BeNil())
			Expect(err).To(And(
				HaveOccurred(),
				MatchError(ContainSubstring("unexpected HTTP status code 403")),
			))
		})
	})
})
