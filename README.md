# AWS subnet exporter

Fetch AWS subnet data and expose it as Prometheus metrics. Why? Because AWS does not expose these in CloudWatch and you don't want to run out of available IP's in your subnets.

This data comprises of ip address count, max and used, for subnets, as well as available contiguous subnet prefixes availables - this is especially important for our EKS nodes.

[This video](https://www.youtube.com/watch?v=RBE3yk2UlYA) is a great explainer for how CNI works and why its important to us.

## Metrics exported

```
aws_subnet_exporter_available_ips
aws_subnet_exporter_available_prefixes Available prefixes in subnets
aws_subnet_exporter_used_prefixes Used prefixes in subnets
aws_subnet_exporter_max_ips Max host IPs in subnet
```

## Assumptions
This service assumes that you subnets have a tag "Name" and that you have exported your AWS access key and secret.

## AWS policy required
You require this policy to your user/role (use roles for best practice) in order to fetch AWS subnet data.
```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "some-sid",
            "Effect": "Allow",
            "Action": "ec2:DescribeSubnets",
            "Resource": "*"
        }
    ]
}
```

## Testing locally

Go run directly:

- Set your AWS creds

```bash
go run cmd/aws-subnet-exporter/main.go
```

This will launch with default params set (all subnets for region and current account)

and then check your metrics outpur

```
curl localhost:8080/metrics
```

## Helm

Helm charts for the exporter are published in [cloud-platform-helm-charts](https://github.com/ministryofjustice/cloud-platform-helm-charts/tree/main/aws-subnet-exporter)