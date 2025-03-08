package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	hashitypes "github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &xsynchcoAWSDataSource{}
	_ datasource.DataSourceWithConfigure = &xsynchcoAWSDataSource{}
)

// NewCoffeesDataSource is a helper function to simplify the provider implementation.
func NewXsynchcoAWSDataSource() datasource.DataSource {
	return &xsynchcoAWSDataSource{}
}

// xsynchcoAWSDataSource is the data source implementation.
type xsynchcoAWSDataSource struct {
	client *ClientS3
}

type bucketModel struct {
	Date        hashitypes.String `tfsdk:"date"`
	Name        hashitypes.String `tfsdk:"name"`
	Tags        []string          `tfsdk:"tags"`
	Description string            `tfsdk:"description"`
}

type awsBucketDataSourceModel struct {
	Buckets []bucketModel `tfsdk:"s3bucket"`
}

// Metadata returns the data source type name.
func (d *xsynchcoAWSDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_aws"
}

// Schema defines the schema for the data source.
func (d *xsynchcoAWSDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"s3bucket": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"date": schema.StringAttribute{
							Computed: true,
						},
						"name": schema.StringAttribute{
							Computed: true,
						},
						"tags": schema.StringAttribute{
							Computed: true,
						},
						"description": schema.StringAttribute{
							Computed: true,
						},
					},
				},
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *xsynchcoAWSDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state awsBucketDataSourceModel

	buckets, err := d.client.S3Client.ListBuckets(context.Background(), nil)
	if err != nil {
		resp.Diagnostics.AddError(
			"unable to read bucket data",
			err.Error(),
		)
		return
	}

	for _, bucket := range buckets.Buckets {
		bucketState := bucketModel{
			Date: hashitypes.StringValue(bucket.CreationDate.Format("2006-01-02 15:04:05")),
			Name: hashitypes.StringValue(*bucket.Name),
		}
		state.Buckets = append(state.Buckets, bucketState)
	}

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Configure adds the provider configured client to the data source.
func (d *xsynchcoAWSDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Add a nil check when handling ProviderData because Terraform
	// sets that data after it calls the ConfigureProvider RPC.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*ClientS3)
	// client, ok := req.ProviderData.(*session.Session)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *session.Session, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}
