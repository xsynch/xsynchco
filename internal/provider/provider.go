package provider

import (
	"context"
	"errors"
	"os"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	hashitypes "github.com/hashicorp/terraform-plugin-framework/types"
    "github.com/hashicorp/terraform-plugin-log/tflog"
	
)


type ClientS3 struct {
	
	S3Client *s3.Client
    Region string 
}

type xsynchco struct {
    Cloud_Provider hashitypes.String `tfsdk:"cloud_provider"`
    Username hashitypes.String  `tfsdk:"username"`
    Password hashitypes.String  `tfsdk:"password"`
    Region hashitypes.String    `tfsdk:"region"`
}


func NewClientS3(region string) (*ClientS3,error){
	ctx := context.Background()
	

	sdkConfig, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return &ClientS3{}, errors.New("error loading aws configuration information")
		
	}
	s3Client := s3.NewFromConfig(sdkConfig)
    
	return &ClientS3{ S3Client: s3Client, Region: region},nil 
}

// Ensure the implementation satisfies the expected interfaces.
var (
    _ provider.Provider = &xsynchProvider{}
)

// New is a helper function to simplify provider server and testing implementation.
func New(version string) func() provider.Provider {
    return func() provider.Provider {
        return &xsynchProvider{
            version: version,
        }
    }
}

// xsynchProvider is the provider implementation.
type xsynchProvider struct {
    // version is set to the provider version on release, "dev" when the
    // provider is built and ran locally, and "test" when running acceptance
    // testing.
    version string
}

// Metadata returns the provider type name.
func (p *xsynchProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
    resp.TypeName = "xsynchco"
    resp.Version = p.version
}

// Schema defines the provider-level schema for configuration data.
func (p *xsynchProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "cloud_provider": schema.StringAttribute{
                Optional: false,
                Required: true,
                Description: "This is the name of the cloud provider you want to use for storage creation",
            },
            "username": schema.StringAttribute{
                Optional: true,
            },
            "password": schema.StringAttribute{
                Optional:  true,
                Sensitive: true,
            },
            "region": schema.StringAttribute{
                Optional: true,
                
            },
        },
    }
}

// Configure prepares a HashiCups API client for data sources and resources.
func (p *xsynchProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
    var xsynchcoConfig xsynchco
    diags := req.Config.Get(ctx, &xsynchcoConfig)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    if xsynchcoConfig.Cloud_Provider.IsUnknown(){
        resp.Diagnostics.AddAttributeError(
            path.Root("cloud_provider"),
            "Unknown Cloud Provider",
            "The provider cannot create a cloud provider client without a cloud provider's name",
        )
    }

    if xsynchcoConfig.Region.IsUnknown(){
        resp.Diagnostics.AddAttributeError(
            path.Root("region"),
            "Unknown Region",
            "Region for the new storage account must be specified",
        )
    }
    region := os.Getenv("S3_REGION")

    if !xsynchcoConfig.Region.IsNull(){
        region = xsynchcoConfig.Region.ValueString()
    }
    
    ctx = tflog.SetField(ctx,"Cloud Provider",xsynchcoConfig.Cloud_Provider.ValueString())
    tflog.Debug(ctx,"Creating AWS Client")
    s3Client,err := NewClientS3(region)
    if err != nil {
        resp.Diagnostics.AddError("unable to create AWS client","An unexpected error occurred creating the AWS client: " + err.Error())
        
    }
    if resp.Diagnostics.HasError(){
        return 
    }
    
    resp.DataSourceData = s3Client
    resp.ResourceData = s3Client

    tflog.Info(ctx,"Configured AWS Client",map[string]any{"success":true})
    
}

// DataSources defines the data sources implemented in the provider.
func (p *xsynchProvider) DataSources(_ context.Context) []func() datasource.DataSource {
    return []func() datasource.DataSource {
        NewXsynchcoAWSDataSource,
    }
}

// Resources defines the resources implemented in the provider.
func (p *xsynchProvider) Resources(_ context.Context) []func() resource.Resource {
    return []func() resource.Resource{
        NewS3Resource,
    }
}
