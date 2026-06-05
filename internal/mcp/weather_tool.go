package mcp

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
)

func init() {
	weatherTool := &Tool{
		Name:        "get_weather",
		Description: "Get current weather information for a city",
		Parameters: []Param{
			{
				Name:        "city",
				Description: "The name of the city to get weather for",
				Type:        "string",
				Required:    true,
			},
		},
		Function: getWeather,
	}

	GetInstance().RegisterTool(weatherTool)
}

func getWeather(params map[string]interface{}) (string, error) {
	city, ok := params["city"].(string)
	if !ok || city == "" {
		return "", fmt.Errorf("city parameter is required")
	}

	apiKey := "YOUR_API_KEY"
	baseURL := "http://api.openweathermap.org/data/2.5/weather"

	paramsURL := url.Values{}
	paramsURL.Set("q", city)
	paramsURL.Set("appid", apiKey)
	paramsURL.Set("units", "metric")
	paramsURL.Set("lang", "zh_cn")

	fullURL := baseURL + "?" + paramsURL.Encode()

	resp, err := http.Get(fullURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch weather: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("weather API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	return string(body), nil
}

func GetWeather(city string) (string, error) {
	return GetInstance().ExecuteTool("get_weather", map[string]interface{}{
		"city": city,
	})
}
