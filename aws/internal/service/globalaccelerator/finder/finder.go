package finder

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/globalaccelerator"
	"github.com/hashicorp/aws-sdk-go-base/tfawserr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// AcceleratorByARN returns the accelerator corresponding to the specified ARN.
// Returns NotFoundError if no accelerator is found.
func AcceleratorByARN(conn *globalaccelerator.GlobalAccelerator, arn string) (*globalaccelerator.Accelerator, error) {
	input := &globalaccelerator.DescribeAcceleratorInput{
		AcceleratorArn: aws.String(arn),
	}

	output, err := conn.DescribeAccelerator(input)

	if tfawserr.ErrCodeEquals(err, globalaccelerator.ErrCodeAcceleratorNotFoundException) {
		return nil, &resource.NotFoundError{
			LastError:   err,
			LastRequest: input,
		}
	}

	if err != nil {
		return nil, err
	}

	if output == nil || output.Accelerator == nil {
		return nil, &resource.NotFoundError{
			Message:     "Empty result",
			LastRequest: input,
		}
	}

	return output.Accelerator, nil
}

// AcceleratorAttributesByARN returns the accelerator corresponding to the specified ARN.
// Returns NotFoundError if no accelerator is found.
func AcceleratorAttributesByARN(conn *globalaccelerator.GlobalAccelerator, arn string) (*globalaccelerator.AcceleratorAttributes, error) {
	input := &globalaccelerator.DescribeAcceleratorAttributesInput{
		AcceleratorArn: aws.String(arn),
	}

	output, err := conn.DescribeAcceleratorAttributes(input)

	if tfawserr.ErrCodeEquals(err, globalaccelerator.ErrCodeAcceleratorNotFoundException) {
		return nil, &resource.NotFoundError{
			LastError:   err,
			LastRequest: input,
		}
	}

	if err != nil {
		return nil, err
	}

	if output == nil || output.AcceleratorAttributes == nil {
		return nil, &resource.NotFoundError{
			Message:     "Empty result",
			LastRequest: input,
		}
	}

	return output.AcceleratorAttributes, nil
}

// EndpointGroupByARN returns the endpoint group corresponding to the specified ARN.
func EndpointGroupByARN(conn *globalaccelerator.GlobalAccelerator, arn string) (*globalaccelerator.EndpointGroup, error) {
	input := &globalaccelerator.DescribeEndpointGroupInput{
		EndpointGroupArn: aws.String(arn),
	}

	output, err := conn.DescribeEndpointGroup(input)
	if err != nil {
		return nil, err
	}

	return output.EndpointGroup, nil
}
