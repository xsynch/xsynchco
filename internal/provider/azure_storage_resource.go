package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	// "github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
	

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &azureStorageResource{}
	_ resource.ResourceWithConfigure = &azureStorageResource{}
)

var (
	resourcesClientFactory *armresources.ClientFactory
	storageClientFactory   *armstorage.ClientFactory
)

var (
	resourceGroupClient *armresources.ResourceGroupsClient
	accountsClient      *armstorage.AccountsClient
	
)


type azureStorageResourceModel struct {
	ID           types.String `tfsdk:"id"`
	Last_Updated types.String `tfsdk:"last_updated"`
	Buckets      []buckets    `tfsdk:"buckets"`
	SubscriptionID types.String `tfsdk:"subscriptionid"`
	ResourceGroupName types.String `tfsdk:"resource_group_name"`
}

type azbuckets struct {
	Date types.String `tfsdk:"date"`
	Name types.String `tfsdk:"name"`
	Tags types.String `tfsdk:"tags"`
}

// NewOrderResource is a helper function to simplify the provider implementation.
func NewAzureStorageResource() resource.Resource {
	return &azureStorageResource{}
}

// s3Resource is the resource implementation.
type azureStorageResource struct {
	client *azureProviderStruct
}

func (r *azureStorageResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*azureProviderStruct)
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
func (r *azureStorageResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_az_storage"
}

// Schema defines the schema for the resource.
func (r *azureStorageResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"subscriptionid": schema.StringAttribute{
				Required: true,
			},
			"resource_group_name": schema.StringAttribute{
				Required: true,
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
func (r *azureStorageResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan azureStorageResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	

	resourcesClientFactory, err = armresources.NewClientFactory(plan.SubscriptionID.String(), r.client.azClient, nil)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating azure resources client factory",
			err.Error(),
		)
		fmt.Println(err)
	}
	resourceGroupClient = resourcesClientFactory.NewResourceGroupsClient()

	storageClientFactory, err = armstorage.NewClientFactory(plan.SubscriptionID.String(), r.client.azClient, nil)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating storage account client",
			err.Error(),
		)
		fmt.Println(err)
	}
	accountsClient = storageClientFactory.NewAccountsClient()
	_, err = resourceGroupClient.CreateOrUpdate(ctx,plan.ResourceGroupName.ValueString(),
		armresources.ResourceGroup{ Location: &r.client.Region,},nil)
	if err != nil {
		resp.Diagnostics.AddError("error creating resource group",err.Error())
		fmt.Println(err)
	}


	for index, item := range plan.Buckets {

		// Create an S3 service client

		// svc := r.client.azClient

		// awsStringBucket := strings.Replace(item.Name.String(), "\"", "", -1)

		// // Create input parameters for the CreateBucket operation

		// input := &s3.CreateBucketInput{

		// 	Bucket: aws.String(awsStringBucket),
		// }

		// Execute the CreateBucket operation

		// _, err := svc.CreateBucket(context.Background(), input)
		storageResponse, err := createStorageAccount(context.Background(),plan.ResourceGroupName.ValueString(),item.Name.ValueString(),r.client.Region)

		if err != nil {

			resp.Diagnostics.AddError(

				"Error creating order",

				"Could not create order, unexpected error: "+err.Error(),
			)

			return

		}
		plan.ID = types.StringValue(*storageResponse.ID)

		// Add tags
		tagValue := strings.Replace(item.Tags.String(), "\"", "", -1)

		_, err = accountsClient.Update(ctx, plan.ResourceGroupName.ValueString(),*storageResponse.Name,armstorage.AccountUpdateParameters{
			Tags: map[string]*string{
				"xsynchco": to.Ptr(tagValue),
			},
		},nil)	
		if err != nil {
			resp.Diagnostics.AddError("error adding tags to storage account",err.Error())
			
			fmt.Println("Error adding tags to the storage account:", err)

			return

		}

		// var tags []awstypes.Tag

		

		// tags = append(tags, awstypes.Tag{

		// 	Key: aws.String("tfkey"),

		// 	Value: aws.String(tagValue),
		// })

		// _, err = svc.PutBucketTagging(ctx, &s3.PutBucketTaggingInput{

		// 	Bucket: aws.String(awsStringBucket),

		// 	Tagging: &awstypes.Tagging{

		// 		TagSet: tags,
		// 	},
		// })

		// if err != nil {

		// 	fmt.Println("Error adding tags to the bucket:", err)

		// 	return

		// }

		fmt.Printf("Bucket %s created successfully\n", item.Name)

		plan.Buckets[index] = buckets{

			Name: types.StringValue(item.Name.ValueString()),

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
func (r *azureStorageResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {

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
func (r *azureStorageResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {

	// Retrieve values from plan

	var plan azureStorageResourceModel

	diags := req.Plan.Get(ctx, &plan)

	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {

		return

	}

	plan.ID = types.StringValue(strconv.Itoa(1))

	for index, item := range plan.Buckets {

		// Create an S3 service client

		svc := r.client.azClient

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
func (r *azureStorageResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {

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

func (r *azureStorageResource) createStorageAccount(ctx context.Context) (*armstorage.Account, error) {

	var accountsClient  *armstorage.AccountsClient

	pollerResp, err := accountsClient.BeginCreate(
		ctx,
		resourceGroupName,
		storageAccountName,
		armstorage.AccountCreateParameters{
			Kind: to.Ptr(armstorage.KindStorageV2),
			SKU: &armstorage.SKU{
				Name: to.Ptr(armstorage.SKUNameStandardLRS),
			},
			Location: to.Ptr(r.client.Region),
			Properties: &armstorage.AccountPropertiesCreateParameters{
				AccessTier: to.Ptr(armstorage.AccessTierCool),
				Encryption: &armstorage.Encryption{
					Services: &armstorage.EncryptionServices{
						File: &armstorage.EncryptionService{
							KeyType: to.Ptr(armstorage.KeyTypeAccount),
							Enabled: to.Ptr(true),
						},
						Blob: &armstorage.EncryptionService{
							KeyType: to.Ptr(armstorage.KeyTypeAccount),
							Enabled: to.Ptr(true),
						},
					},
					KeySource: to.Ptr(armstorage.KeySourceMicrosoftStorage),
				},
			},
		}, nil)
	if err != nil {
		return nil, err
	}
	resp, err := pollerResp.PollUntilDone(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &resp.Account, nil
}
