package exporter

import (
	"fmt"
	"strings"
)

type DataSource struct {
	Name         string
	VpnFilter    string
	ItemFilter   string
	MetricFilter []string
}

func (dataSource DataSource) String() string {
	return fmt.Sprintf("%s=%s|%s|%s", dataSource.Name, dataSource.VpnFilter, dataSource.ItemFilter, strings.Join(dataSource.MetricFilter, ","))
}
