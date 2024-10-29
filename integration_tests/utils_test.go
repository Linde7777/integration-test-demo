ackage integration_tests

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"net/url"
	"testing"
)

type ComplexReq struct {
	MaxDistanceKm int    `form:"max_distance_km"`
	MinAge        int    `form:"min_age"`
	MaxAge        int    `form:"max_age"`
	Gender        string `form:"gender"`
}

func TestStructToQueryString(t *testing.T) {
	req := ComplexReq{
		MaxDistanceKm: 100,
		MinAge:        18,
		MaxAge:        30,
		Gender:        "female",
	}

	queryString := structToQueryString(req)
	fmt.Println(queryString)

	// 解码并验证结果
	values, _ := url.ParseQuery(queryString)
	require.Equal(t, "100", values.Get("max_distance_km"))
	require.Equal(t, "18", values.Get("min_age"))
	require.Equal(t, "30", values.Get("max_age"))
	require.Equal(t, "female", values.Get("gender"))
}
