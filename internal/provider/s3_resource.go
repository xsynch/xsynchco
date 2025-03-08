package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	awstypes "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &s3Resource{}
	_ resource.ResourceWithConfigure = &s3Resource{}
)

type s3ResourceModel struct {
	ID           types.String `tfsdk:"id"`
	Last_Updated types.String `tfsdk:"last_updated"`
	Buckets      []buckets    `tfsdk:"buckets"`
}

type buckets struct {
	Date types.String `tfsdk:"date"`
	Name types.String `tfsdk:"name"`
	Tags types.String `tfsdk:"tags"`
}

// NewOrderResource is a helper function to simplify the provider implementation.
func NewS3Resource() resource.Resource {
	return &s3Resource{}
}

// s3Resource is the resource implementation.
type s3Resource struct {
	client *ClientS3
}

func (r *s3Resource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*ClientS3)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected DataSource Configure Type",
			fmt.Sprintf("Expected *ClientS3, got: %T. Please report this issue to the developer", req.ProviderData),
		)
		return
	}
	r.client = client

}

// Metadata returns the resource type name.
func (r *s3Resource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_s3_storage"
}

// Schema defines the schema for the resource.
func (r *s3Resource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"last_updated": schema.StringAttribute{
				Computed: true,
			},
			"buckets": schema.ListNestedAttribute{
				Required: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"date": schema.StringAttribute{
							Computed: true,
						},
						"name": schema.StringAttribute{
							Required: true,
						},
						"tags": schema.StringAttribute{
							Required: true,
						},
					},
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *s3Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan s3ResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = types.StringValue(strconv.Itoa(1))
	for index, item := range plan.Buckets {

		// Create an S3 service client

		svc := r.client.S3Client

		awsStringBucket := strings.Replace(item.Name.String(), "\"", "", -1)

		// Create input parameters for the CreateBucket operation

		input := &s3.CreateBucketInput{

			Bucket: aws.String(awsStringBucket),
		}

		// Execute the CreateBucket operation

		_, err := svc.CreateBucket(context.Background(), input)

		if err != nil {

			resp.Diagnostics.AddError(

				"Error creating order",

				"Could not create order, unexpected error: "+err.Error(),
			)

			return

		}

		// Add tags

		var tags []awstypes.Tag

		tagValue := strings.Replace(item.Tags.String(), "\"", "", -1)

		tags = append(tags, awstypes.Tag{

			Key: aws.String("tfkey"),

			Value: aws.String(tagValue),
		})

		_, err = svc.PutBucketTagging(ctx, &s3.PutBucketTaggingInput{

			Bucket: aws.String(awsStringBucket),

			Tagging: &awstypes.Tagging{

				TagSet: tags,
			},
		})

		if err != nil {

			fmt.Println("Error adding tags to the bucket:", err)

			return

		}

		fmt.Printf("Bucket %s created successfully\n", item.Name)

		plan.Buckets[index] = buckets{

			Name: types.StringValue(awsStringBucket),

			Date: types.StringValue(time.Now().Format(time.RFC850)),

			Tags: types.StringValue(tagValue),
		}

	}

	plan.Last_Updated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)

	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {

		return

	}
}

// Read refreshes the Terraform state with the latest data.
func (r *s3Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {

	// Get current state

	var state s3ResourceModel

	diags := req.State.Get(ctx, &state)

	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {

		return

	}

	for _, item := range state.Buckets {

		awsStringBucket := strings.Replace(item.Name.String(), "\"", "", -1)

		svc := r.client.S3Client

		params := &s3.HeadBucketInput{

			Bucket: aws.String(awsStringBucket),
		}

		_, err := svc.HeadBucket(ctx, params)

		if err != nil {

			tflog.Error(ctx, fmt.Sprintf("error getting bucket information: %s", err))
			return

		}

	}

	// Set refreshed state

	diags = resp.State.Set(ctx, &state)

	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {

		return

	}

}

// Update updates the resource and sets the updated Terraform state on success.
func (r *s3Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {

	// Retrieve values from plan

	var plan s3ResourceModel

	diags := req.Plan.Get(ctx, &plan)

	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {

		return

	}

	plan.ID = types.StringValue(strconv.Itoa(1))

	for index, item := range plan.Buckets {

		// Create an S3 service client

		svc := r.client.S3Client

		awsStringBucket := strings.Replace(item.Name.String(), "\"", "", -1)

		// Add tags

		var tags []awstypes.Tag

		tagValue := strings.Replace(item.Tags.String(), "\"", "", -1)

		tags = append(tags, awstypes.Tag{

			Key: aws.String("tfkey"),

			Value: aws.String(tagValue),
		})

		_, err := svc.PutBucketTagging(context.Background(), &s3.PutBucketTaggingInput{

			Bucket: aws.String(awsStringBucket),

			Tagging: &awstypes.Tagging{

				TagSet: tags,
			},
		})

		if err != nil {

			fmt.Println("Error adding tags to the bucket:", err)

			return

		}

		plan.Buckets[index] = buckets{

			Name: types.StringValue(strings.Replace(awsStringBucket, "\"", "", -1)),

			Date: types.StringValue(time.Now().Format(time.RFC850)),

			Tags: types.StringValue(strings.Replace(tagValue, "\"", "", -1)),
		}

	}

	plan.Last_Updated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)

	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {

		return

	}

}

// Delete deletes the resource and removes the Terraform state on success.
func (r *s3Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {

	// Retrieve values from state

	var state s3ResourceModel

	diags := req.State.Get(ctx, &state)

	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {

		return

	}

	for _, item := range state.Buckets {

		svc := r.client.S3Client

		input := &s3.DeleteBucketInput{

			Bucket: aws.String(strings.Replace(item.Name.String(), "\"", "", -1)),
		}

		_, err := svc.DeleteBucket(context.Background(), input)

		if err != nil {

			// log.Fatalf("failed to delete bucket, %v", err)
			tflog.Error(ctx, fmt.Sprintf("failed to delete bucket: %v", err), map[string]any{"success": false})

		}

	}

}
