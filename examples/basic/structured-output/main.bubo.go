// Code generated by bubo-tool-gen. DO NOT EDIT.
// This file was generated for bubo:agentTool marker comments in the source code.

package main

import "github.com/casualjim/bubo/tool"

// Get the current weather in a given location. Location MUST be a city.
var getWeatherTool = tool.Must(getWeather, tool.Name("getWeather"), tool.Description("Get the current weather in a given location. Location MUST be a city."), tool.Parameters("location"))
