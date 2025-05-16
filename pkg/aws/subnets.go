package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/ministryofjustice/aws-subnet-exporter/pkg/utils"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type Subnet struct {
	Name              string
	SubnetID          string
	VPCID             string
	CIDRBlock         string
	AZ                string
	AvailableIPs      float64
	MaxIPs            float64
	UsedPrefixes      int
	AvailablePrefixes []string
}

func GetSubnets(client *ec2.Client, filter string) ([]Subnet, error) {
	log.Debug("Describing subnets")
	nameIdentifier := "tag:Name"
	resp, err := client.DescribeSubnets(context.TODO(), &ec2.DescribeSubnetsInput{
		Filters: []types.Filter{{
			Name:   &nameIdentifier,
			Values: []string{filter},
		}},
	})
	if err != nil {
		log.Debug("Failed to describe subnets")
		return nil, errors.Wrap(err, "cannot describe subnets")
	}

	var subnets []Subnet
	for _, v := range resp.Subnets {
		subnet, err := processSubnet(client, v)
		if err != nil {
			return nil, err
		}
		subnets = append(subnets, subnet)
	}
	return subnets, nil
}

func processSubnet(ec2Client *ec2.Client, v types.Subnet) (Subnet, error) {
	log.Debugf("Processing subnet: %s", *v.SubnetId)
	subnet := Subnet{
		Name:         utils.GetNameFromTags(v.Tags),
		SubnetID:     *v.SubnetId,
		VPCID:        *v.VpcId,
		CIDRBlock:    *v.CidrBlock,
		AZ:           *v.AvailabilityZone,
		AvailableIPs: float64(*v.AvailableIpAddressCount),
	}

	describeSubnetsOutput := &ec2.DescribeSubnetsOutput{
		Subnets: []types.Subnet{v},
	}

	details, err := utils.EnrichSubnetData(describeSubnetsOutput)
	if err != nil {
		return Subnet{}, errors.Wrap(err, "unable to get subnet details")
	}

	subnet.MaxIPs = float64(details.TotalIPs)

	networkInterfacesOutput, err := utils.DescribeNetworkInterfacesBySubnetID(context.TODO(), ec2Client, *v.SubnetId)
	if err != nil {
		return Subnet{}, errors.Wrap(err, "unable to describe network interfaces")
	}
	
	prefixesInUse, ipsInUse, err := utils.EnrichIPsAndPrefixes(networkInterfacesOutput, details)
	if err != nil {
		return Subnet{}, errors.Wrap(err, "unable to get IPs and prefixes")
	}

	utils.CalculatePrefixes(details, prefixesInUse, ipsInUse)

	subnet.UsedPrefixes = details.PrefixesInUse
	subnet.AvailablePrefixes = details.AvailablePrefixes

	return subnet, nil
}