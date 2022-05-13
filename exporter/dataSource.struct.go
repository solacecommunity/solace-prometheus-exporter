package exporter

import "fmt"

type DataSource struct {
	Name       string
	VpnFilter  string
	ItemFilter string
}

func (dataSource DataSource) String() string {
	return fmt.Sprintf("%s=%s|%s", dataSource.Name, dataSource.VpnFilter, dataSource.ItemFilter)
}
