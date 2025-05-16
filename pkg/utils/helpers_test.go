package utils

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func TestCalculateMaxIPs(t *testing.T) {
	tests := []struct {
		name      string
		cidr      string
		want      float64
		expectErr bool
	}{
		{
			name:      "Valid /24 CIDR",
			cidr:      "172.16.0.0/24",
			want:      254,
			expectErr: false,
		},
		{
			name:      "Valid /16 CIDR",
			cidr:      "172.16.0.0/16",
			want:      65534,
			expectErr: false,
		},
		{
			name:      "Valid /30 CIDR",
			cidr:      "172.16.0.0/30",
			want:      2,
			expectErr: false,
		},
		{
			name:      "Valid /32 CIDR (single IP)",
			cidr:      "192.168.1.1/32",
			want:      -1,
			expectErr: false,
		},
		{
			name:      "Invalid CIDR format",
			cidr:      "172.16.0.0",
			want:      0,
			expectErr: true,
		},
		{
			name:      "Invalid IP in CIDR",
			cidr:      "999.999.999.999/24",
			want:      0,
			expectErr: true,
		},
		{
			name:      "Empty string",
			cidr:      "",
			want:      0,
			expectErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CalculateMaxIPs(tt.cidr)
			if (err != nil) != tt.expectErr {
				t.Errorf("CalculateMaxIPs() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if !tt.expectErr && got != tt.want {
				t.Errorf("CalculateMaxIPs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSplitCIDR(t *testing.T) {
	tests := []struct {
		name      string
		cidr      string
		wantIP    string
		wantMask  int
		expectErr bool
	}{
		{
			name:      "Valid CIDR",
			cidr:      "172.16.0.0/24",
			wantIP:    "172.16.0.0",
			wantMask:  24,
			expectErr: false,
		},
		{
			name:      "Valid CIDR with different mask",
			cidr:      "172.16.0.0/16",
			wantIP:    "172.16.0.0",
			wantMask: 16,
			expectErr: false,
		},
		{
			name:      "Invalid CIDR format",
			cidr:      "172.16.0.0",
			wantIP:    "",
			wantMask:  0,
			expectErr: true,
		},
		{
			name:      "Invalid CIDR with non-numeric mask",
			cidr:      "172.16.0.0/ab",
			wantIP:    "",
			wantMask:  0,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotIP, gotMask, err := splitCIDR(tt.cidr)
			if (err != nil) != tt.expectErr {
				t.Errorf("splitCIDR() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if !tt.expectErr && (gotIP != tt.wantIP || gotMask != tt.wantMask) {
				t.Errorf("splitCIDR() = %v/%v, want %v/%v", gotIP, gotMask, tt.wantIP, tt.wantMask)
			}
		})
	}
}

func TestSplitIP(t *testing.T) {
	tests := []struct {
		name      string
		ip        string
		want      []int
		expectErr bool
	}{
		{
			name:      "Valid IP",
			ip:        "172.16.0.125",
			want:      []int{172, 16, 0, 125},
			expectErr: false,
		},
		{
			name:	  "Invalid IP format short",
			ip:        "172.16.0",
			want:      nil,
			expectErr: true,
		},
		{
			name:      "Invalid IP format long",
			ip:        "172.16.0.1.1",
			want:      nil,
			expectErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := splitIP(tt.ip)
			if (err != nil) != tt.expectErr {
				t.Errorf("splitIP() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if !tt.expectErr && !equalSlices(got, tt.want) {
				t.Errorf("splitIP() = %v, want %v", got, tt.want)
			}
		})
	}
}

// helper function for comparing ip slices
func equalSlices(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestEnrichSubnetData(t *testing.T) {
    tests := []struct {
        name      string
        input     *ec2.DescribeSubnetsOutput
        want      *SubnetDetails
        expectErr bool
    }{
        {
            name: "Valid subnet data",
            input: &ec2.DescribeSubnetsOutput{
                Subnets: []types.Subnet{
                    {
                        CidrBlock: aws.String("172.16.1.0/24"),
                    },
                },
            },
            want: &SubnetDetails{
                SubnetCIDR:      "172.16.1.0/24",
                SubnetMask:      24,
                TotalIPs:        256,
                CIDRFirstDigit:  172,
                CIDRSecondDigit: 16,
                CIDRThirdDigit:  1,
                CIDRLastDigit:   0,
            },
            expectErr: false,
        },
        {
            name: "Empty subnets",
            input: &ec2.DescribeSubnetsOutput{
                Subnets: []types.Subnet{},
            },
            want:      nil,
            expectErr: true,
        },
        {
            name: "Invalid CIDR in subnet",
            input: &ec2.DescribeSubnetsOutput{
                Subnets: []types.Subnet{
                    {
                        CidrBlock: aws.String("172.16.1.0"),
                    },
                },
            },
            want:      nil,
            expectErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := EnrichSubnetData(tt.input)
            if (err != nil) != tt.expectErr {
                t.Errorf("EnrichSubnetData() error = %v, expectErr %v", err, tt.expectErr)
                return
            }
            if !tt.expectErr && got != nil && tt.want != nil {
                if got.SubnetCIDR != tt.want.SubnetCIDR ||
                    got.SubnetMask != tt.want.SubnetMask ||
                    got.TotalIPs != tt.want.TotalIPs ||
                    got.CIDRFirstDigit != tt.want.CIDRFirstDigit ||
                    got.CIDRSecondDigit != tt.want.CIDRSecondDigit ||
                    got.CIDRThirdDigit != tt.want.CIDRThirdDigit ||
                    got.CIDRLastDigit != tt.want.CIDRLastDigit {
                    t.Errorf("EnrichSubnetData() = %+v, want %+v", got, tt.want)
					// output got and tt.want for debugging
					fmt.Printf("got: %+v\n", got)
					fmt.Printf("want: %+v\n", tt.want)

                }
            }
        })
    }
}

func TestEnrichIPsAndPrefixes(t *testing.T) {
    tests := []struct {
        name         string
        input        *ec2.DescribeNetworkInterfacesOutput
        details      *SubnetDetails
        wantIPs      map[string]bool
        wantPrefixes map[string]bool
        expectErr    bool
    }{
        {
            name: "Valid network interfaces",
            input: &ec2.DescribeNetworkInterfacesOutput{
                NetworkInterfaces: []types.NetworkInterface{
                    {
						PrivateIpAddresses: []types.NetworkInterfacePrivateIpAddress{
							{PrivateIpAddress: aws.String("172.16.1.125")},
							{PrivateIpAddress: aws.String("172.16.1.126")},
						},
                        Ipv4Prefixes: []types.Ipv4PrefixSpecification{
                            {Ipv4Prefix: aws.String("172.16.1.112/28")},
                        },
                    },
                },
            },
            details: &SubnetDetails{
                SubnetCIDR:      "172.16.1.0/24",
                SubnetMask:      24,
                TotalIPs:        256,
                CIDRFirstDigit:  172,
                CIDRSecondDigit: 16,
                CIDRThirdDigit:  1,
                CIDRLastDigit:   0,
            },
            wantIPs: map[string]bool{
                "172.16.1.125": true,
                "172.16.1.126": true,
            },
            wantPrefixes: map[string]bool{
                "172.16.1.112/28": true,
            },
            expectErr: false,
        },
        {
            name: "No network interfaces",
            input: &ec2.DescribeNetworkInterfacesOutput{
                NetworkInterfaces: []types.NetworkInterface{},
            },
            details: &SubnetDetails{
                SubnetCIDR:      "172.16.1.0/24",
                SubnetMask:      24,
                TotalIPs:        256,
                CIDRFirstDigit:  172,
                CIDRSecondDigit: 16,
                CIDRThirdDigit:  1,
                CIDRLastDigit:   0,
            },
            wantIPs:      map[string]bool{},
            wantPrefixes: map[string]bool{},
            expectErr:    false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            gotPrefixes, gotIPs, err := EnrichIPsAndPrefixes(tt.input, tt.details)
            if (err != nil) != tt.expectErr {
                t.Errorf("EnrichIPsAndPrefixes() error = %v, expectErr %v", err, tt.expectErr)
                return
            }
            if !mapsEqual(gotIPs, tt.wantIPs) {
                t.Errorf("EnrichIPsAndPrefixes() gotIPs = %v, want %v", gotIPs, tt.wantIPs)
            }
            if !mapsEqual(gotPrefixes, tt.wantPrefixes) {
                t.Errorf("EnrichIPsAndPrefixes() gotPrefixes = %v, want %v", gotPrefixes, tt.wantPrefixes)
            }
        })
    }
}

// helper for comparing maps
func mapsEqual(a, b map[string]bool) bool {
    if len(a) != len(b) {
        return false
    }
    for k, v := range a {
        if b[k] != v {
            return false
        }
    }
    return true
}