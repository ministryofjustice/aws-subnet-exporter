package utils

import (
	"context"
	"fmt"
	"math"
	"net"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/pkg/errors"
)

type SubnetDetails struct {
    SubnetCIDR          string
    SubnetMask          int
    TotalIPs            int
    CIDRFirstDigit      int
    CIDRSecondDigit     int
    CIDRThirdDigit      int
    CIDRLastDigit       int
    InterfacesInUse     int
    AllocatedIPs        int
    FreeIPs             int
    MaxPrefixes         int
    PrefixesInUse       int
    AvailablePrefixes   []string
}

func CalculateMaxIPs(cidr string) (float64, error) {
    _, IPNet, err := net.ParseCIDR(cidr)
    if err != nil {
        return 0, errors.Wrap(err, "cannot parse CIDR block")
    }

    ones, bits := IPNet.Mask.Size()
    totalIPs := math.Pow(2, float64(bits-ones))
    return totalIPs - 2, nil
}

func splitCIDR(cidr string) (string, int, error) {
    parts := strings.Split(cidr, "/")
    if len(parts) != 2 {
        return "", 0, fmt.Errorf("invalid CIDR format: %s", cidr)
    }
    mask, err := strconv.Atoi(parts[1])
    if err != nil {
        return "", 0, fmt.Errorf("invalid subnet mask: %w", err)
    }
    return parts[0], mask, nil
}

func splitIP(ip string) ([]int, error) {
    parts := strings.Split(ip, ".")
    if len(parts) != 4 {
        return nil, fmt.Errorf("invalid IP format: %s", ip)
    }
    result := make([]int, 4)
    for i, part := range parts {
        val, err := strconv.Atoi(part)
        if err != nil {
            return nil, fmt.Errorf("invalid IP part: %w", err)
        }
        result[i] = val
    }
    return result, nil
}

func DescribeSubnetByID(ctx context.Context, ec2Client *ec2.Client, subnetID string) (*ec2.DescribeSubnetsOutput, error) {
    output, err := ec2Client.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
        SubnetIds: []string{subnetID},
    })
    if err != nil {
        return nil, fmt.Errorf("failed to describe subnets: %w", err)
    }
    return output, nil
}

func EnrichSubnetData(output *ec2.DescribeSubnetsOutput) (*SubnetDetails, error) {
    if len(output.Subnets) == 0 {
        return nil, fmt.Errorf("no subnets found in output")
    }
    subnetCIDR := aws.ToString(output.Subnets[0].CidrBlock)
    ip, mask, err := splitCIDR(subnetCIDR)
    if err != nil {
        return nil, err
    }

    ipParts, err := splitIP(ip)
    if err != nil {
        return nil, err
    }

    totalIPs := int(math.Pow(2, float64(32-mask)))

    return &SubnetDetails{
        SubnetCIDR:      subnetCIDR,
        SubnetMask:      mask,
        TotalIPs:        totalIPs,
        CIDRFirstDigit:  ipParts[0],
        CIDRSecondDigit: ipParts[1],
        CIDRThirdDigit:  ipParts[2],
        CIDRLastDigit:   ipParts[3],
    }, nil
}

func DescribeNetworkInterfacesBySubnetID(ctx context.Context, ec2Client *ec2.Client, subnetID string) (*ec2.DescribeNetworkInterfacesOutput, error) {
    output, err := ec2Client.DescribeNetworkInterfaces(ctx, &ec2.DescribeNetworkInterfacesInput{
        Filters: []types.Filter{
            {
                Name:   aws.String("subnet-id"),
                Values: []string{subnetID},
            },
        },
    })
    if err != nil {
        return nil, fmt.Errorf("failed to describe network interfaces: %w", err)
    }
    return output, nil
}

func EnrichIPsAndPrefixes(output *ec2.DescribeNetworkInterfacesOutput, details *SubnetDetails) (map[string]bool, map[string]bool, error) {
    prefixesInUse := make(map[string]bool)
    ipsInUse := make(map[string]bool)
    ipsPerPrefix := 16

    for _, iface := range output.NetworkInterfaces {
        details.InterfacesInUse++
        for _, privateIP := range iface.PrivateIpAddresses {
            ipsInUse[aws.ToString(privateIP.PrivateIpAddress)] = true
        }
        for _, prefix := range iface.Ipv4Prefixes {
            prefixesInUse[aws.ToString(prefix.Ipv4Prefix)] = true
        }
    }

    details.PrefixesInUse = len(prefixesInUse)
    details.AllocatedIPs = (details.PrefixesInUse * ipsPerPrefix) + len(ipsInUse) + 5
    details.MaxPrefixes = details.TotalIPs / ipsPerPrefix

    return prefixesInUse, ipsInUse, nil
}

func CalculatePrefixes(details *SubnetDetails, prefixesInUse map[string]bool, ipsInUse map[string]bool) {
    ipsPerPrefix := 16
    availablePrefixes := []string{}

    baseFirstDigit := details.CIDRFirstDigit
    baseSecondDigit := details.CIDRSecondDigit
    baseThirdDigit := details.CIDRThirdDigit
    baseLastDigit := details.CIDRLastDigit

    for i := 1; i <= details.MaxPrefixes; i++ {

        baseLastDigit += ipsPerPrefix
        if baseLastDigit > 255 {
            baseThirdDigit++
            baseLastDigit = 0
        }

        prefix := fmt.Sprintf("%d.%d.%d.%d/%d", baseFirstDigit, baseSecondDigit, baseThirdDigit, baseLastDigit, 28)

        if prefixesInUse[prefix] {
            continue
        }

        isAvailable := true
        for j := 0; j < ipsPerPrefix; j++ {
            ip := fmt.Sprintf("%d.%d.%d.%d", baseFirstDigit, baseSecondDigit, baseThirdDigit, baseLastDigit+j)
            if ipsInUse[ip] {
                isAvailable = false
                break
            }
        }

        if isAvailable {
            availablePrefixes = append(availablePrefixes, prefix)
        }
    }

    details.AvailablePrefixes = availablePrefixes
    details.FreeIPs = details.TotalIPs - details.AllocatedIPs

}

func GetNameFromTags(tags []types.Tag) string {
	for _, v := range tags {
		if *v.Key == "Name" {
			return *v.Value
		}
	}
	return "No name tag found"
}