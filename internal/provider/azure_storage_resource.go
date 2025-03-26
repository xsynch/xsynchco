package provider

import (
	"context"
	"fmt"

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
	
	Last_Updated types.String `tfsdk:"last_updated"`
	StorageAccount      []azbuckets    `tfsdk:"storage_accounts"`
	// StorageAccount      []armstorage.Account    `tfsdk:"storage_account"`
	SubscriptionID types.String `tfsdk:"subscriptionid"`
	ResourceGroupName types.String `tfsdk:"resource_group_name"`
}

type azbuckets struct {
	ID types.String `tfsdk:"id"`
	Date types.String `tfsdk:"date"`
	Name types.String `tfsdk:"name"`
	Tags types.String `tfsdk:"tags"`
}

// NewOrderResource is a helper function to simplify the provider implementation.
func NewAzureStorageResource() resource.Resource {
	return &azureStorageResource{}
}

// azure storage account is the resource implementation.
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
			"subscriptionid": schema.StringAttribute{
				Required: true,
			},
			"resource_group_name": schema.StringAttribute{
				Required: true,
			},	
			"last_updated": schema.StringAttribute{
				Computed: true,
			},
			"storage_accounts": schema.ListNestedAttribute{
				Required: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed: true,
						},
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
	

	resourcesClientFactory, err = armresources.NewClientFactory(plan.SubscriptionID.ValueString(), r.client.azClient, nil)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating azure resources client factory",
			err.Error(),
		)
		fmt.Println(err)
	}
	resourceGroupClient = resourcesClientFactory.NewResourceGroupsClient()

	storageClientFactory, err = armstorage.NewClientFactory(plan.SubscriptionID.ValueString(), r.client.azClient, nil)
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
		return 
	}


	for index, item := range plan.StorageAccount {


		storageResponse, err := createStorageAccount(context.Background(),plan.ResourceGroupName.ValueString(),item.Name.ValueString(),r.client.Region)

		if err != nil {

			resp.Diagnostics.AddError(

				"Error creating order",

				"Could not create order, unexpected error: "+err.Error(),
			)

			return

		}
		// plan.ID = types.StringValue(*storageResponse.ID)

		// Add tags
		tagValue := strings.Replace(item.Tags.ValueString(), "\"", "", -1)

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

	

		fmt.Printf("Bucket %s created successfully\n", item.Name)

		plan.StorageAccount[index] = azbuckets{
			ID: types.StringValue(*storageResponse.ID),

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

	var state azureStorageResourceModel

	diags := req.State.Get(ctx, &state)

	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {

		return

	}

	resourcesClientFactory, err = armresources.NewClientFactory(state.SubscriptionID.String(), r.client.azClient, nil)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating azure resources client factory",
			err.Error(),
		)
		fmt.Println(err)
	}
	resourceGroupClient = resourcesClientFactory.NewResourceGroupsClient()

	storageClientFactory, err = armstorage.NewClientFactory(state.SubscriptionID.String(), r.client.azClient, nil)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating storage account client",
			err.Error(),
		)
		fmt.Println(err)
	}
	accountsClient = storageClientFactory.NewAccountsClient()

	//overwrite whatever is is the state with the current values
	state.StorageAccount = make([]azbuckets,0)
	storageAccounts := make([]*armstorage.Account,0)


		//need to get a status of all storage accounts within the resource group

	listAccounts := accountsClient.NewListPager(nil)
	for listAccounts.More(){
		pageResponse, err := listAccounts.NextPage(ctx)
		if err != nil {
			tflog.Error(ctx,fmt.Sprintf("Error listing storage accounts: %s",err.Error()))
			fmt.Println("Error listing storage accounts ",err.Error())
			return 
		}
		storageAccounts = append(storageAccounts,pageResponse.AccountListResult.Value...)

	}
	for _ ,storageAccount := range storageAccounts{

		state.StorageAccount = append(state.StorageAccount, azbuckets{
			ID: types.StringPointerValue(storageAccount.ID),
			Name: types.StringPointerValue(storageAccount.Name),
			Date: types.StringValue(storageAccount.Properties.CreationTime.UTC().String()),
			Tags: types.StringValue(fmt.Sprintf("%v",storageAccount.Tags)),
		})
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
	subscriptionId := plan.SubscriptionID.ValueString()


	resourcesClientFactory, err = armresources.NewClientFactory(subscriptionId, r.client.azClient, nil)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating azure resources client factory",
			err.Error(),
		)
		fmt.Println(err)
		return 
	}
	resourceGroupClient = resourcesClientFactory.NewResourceGroupsClient()

	storageClientFactory, err = armstorage.NewClientFactory(subscriptionId, r.client.azClient, nil)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating storage account client",
			err.Error(),
		)
		fmt.Println(err)
		return 
	}
	accountsClient = storageClientFactory.NewAccountsClient()

	// plan.ID = types.StringValue(strconv.Itoa(1))

	for index, item := range plan.StorageAccount {

		// Create an S3 service client

		// svc := r.client.azClient

		storageAccountName := strings.Replace(item.Name.String(), "\"", "", -1)

				// Add tags
		tagValue := strings.Replace(item.Tags.ValueString(), "\"", "", -1)

		azClientUpdateResp, err := accountsClient.Update(ctx, plan.ResourceGroupName.ValueString(),storageAccountName,armstorage.AccountUpdateParameters{
					Tags: map[string]*string{
						"xsynchco": to.Ptr(tagValue),
				},
				},nil)	
		if err != nil {
			resp.Diagnostics.AddError("error adding tags to storage account",err.Error())
			fmt.Println("Error adding tags to the storage account:", err)
			return
		
		}
		

		plan.StorageAccount[index] = azbuckets{
			ID: types.StringPointerValue(azClientUpdateResp.ID),

			Name: types.StringValue(strings.Replace(storageAccountName, "\"", "", -1)),

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

	var state azureStorageResourceModel

	diags := req.State.Get(ctx, &state)

	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {

		return

	}
	subscriptionId := state.SubscriptionID.ValueString()
	


	resourcesClientFactory, err = armresources.NewClientFactory(subscriptionId, r.client.azClient, nil)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating azure resources client factory",
			err.Error(),
		)
		fmt.Println(err)
		return 
	}
	resourceGroupClient = resourcesClientFactory.NewResourceGroupsClient()

	storageClientFactory, err = armstorage.NewClientFactory(subscriptionId, r.client.azClient, nil)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating storage account client",
			err.Error(),
		)
		fmt.Println(err)
		return 
	}
	accountsClient = storageClientFactory.NewAccountsClient()

	for _, item := range state.StorageAccount {

		_,err = accountsClient.Delete(ctx,r.client.resourceGroupName,item.Name.ValueString(),nil)
		
		if err != nil {
			tflog.Error(ctx,fmt.Sprintf("Error deleteing %s due to: %s",item.Name.ValueString(),err.Error()),map[string]any{"success":false})
			return
		}
		tflog.Info(ctx,fmt.Sprintf("%s deleted successfully\n",item.Name.ValueString()),map[string]any{"success":true})


	}
	 
	

}


// func createAccountsClient(){
// 	resourceGroupClient = resourcesClientFactory.NewResourceGroupsClient()

// 	storageClientFactory, err = armstorage.NewClientFactory(plan.SubscriptionID.String(), r.client.azClient, nil)
// 	if err != nil {

// 		fmt.Println(err)
// 	}
// 	accountsClient = storageClientFactory.NewAccountsClient()
// 	_, err = resourceGroupClient.CreateOrUpdate(ctx,plan.ResourceGroupName.ValueString(),
// 		armresources.ResourceGroup{ Location: &r.client.Region,},nil)
// 	if err != nil {
// 		resp.Diagnostics.AddError("error creating resource group",err.Error())
// 		fmt.Println(err)
// 		return 
// 	}

// }

// func (r *azureStorageResource) createStorageAccount(ctx context.Context) (*armstorage.Account, error) {

// 	var accountsClient  *armstorage.AccountsClient

// 	pollerResp, err := accountsClient.BeginCreate(
// 		ctx,
// 		resourceGroupName,
// 		storageAccountName,
// 		armstorage.AccountCreateParameters{
// 			Kind: to.Ptr(armstorage.KindStorageV2),
// 			SKU: &armstorage.SKU{
// 				Name: to.Ptr(armstorage.SKUNameStandardLRS),
// 			},
// 			Location: to.Ptr(r.client.Region),
// 			Properties: &armstorage.AccountPropertiesCreateParameters{
// 				AccessTier: to.Ptr(armstorage.AccessTierCool),
// 				Encryption: &armstorage.Encryption{
// 					Services: &armstorage.EncryptionServices{
// 						File: &armstorage.EncryptionService{
// 							KeyType: to.Ptr(armstorage.KeyTypeAccount),
// 							Enabled: to.Ptr(true),
// 						},
// 						Blob: &armstorage.EncryptionService{
// 							KeyType: to.Ptr(armstorage.KeyTypeAccount),
// 							Enabled: to.Ptr(true),
// 						},
// 					},
// 					KeySource: to.Ptr(armstorage.KeySourceMicrosoftStorage),
// 				},
// 			},
// 		}, nil)
// 	if err != nil {
// 		return nil, err
// 	}
// 	resp, err := pollerResp.PollUntilDone(ctx, nil)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &resp.Account, nil
// }
