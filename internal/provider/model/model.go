package model

import "github.com/hashicorp/terraform-plugin-framework/types"

type RemoteFileDataSourceModel struct {
	AllowMissing  types.Bool   `tfsdk:"allow_missing"`
	Contents      types.String `tfsdk:"contents"`
	Host          types.String `tfsdk:"host"`
	HostKey       types.String `tfsdk:"host_key"`
	LastModified  types.String `tfsdk:"last_modified"`
	Password      types.String `tfsdk:"password"`
	Path          types.String `tfsdk:"path"`
	Port          types.Int64  `tfsdk:"port"`
	PrivateKey    types.String `tfsdk:"private_key"`
	Size          types.Int64  `tfsdk:"size"`
	Timeout       types.String `tfsdk:"timeout"`
	Triggers      types.Map    `tfsdk:"triggers"`
	User          types.String `tfsdk:"user"`
	ID            types.String `tfsdk:"id"`
	RetryCount    types.Int64  `tfsdk:"retry_count"`
	RetryInterval types.String `tfsdk:"retry_interval"`
}

func (r *RemoteFileDataSourceModel) GetAllowMissing() types.Bool    { return r.AllowMissing }
func (r *RemoteFileDataSourceModel) GetContents() types.String      { return r.Contents }
func (r *RemoteFileDataSourceModel) GetHost() types.String          { return r.Host }
func (r *RemoteFileDataSourceModel) GetHostKey() types.String       { return r.HostKey }
func (r *RemoteFileDataSourceModel) GetLastModified() types.String  { return r.LastModified }
func (r *RemoteFileDataSourceModel) GetPassword() types.String      { return r.Password }
func (r *RemoteFileDataSourceModel) GetPath() types.String          { return r.Path }
func (r *RemoteFileDataSourceModel) GetPort() types.Int64           { return r.Port }
func (r *RemoteFileDataSourceModel) GetPrivateKey() types.String    { return r.PrivateKey }
func (r *RemoteFileDataSourceModel) GetSize() types.Int64           { return r.Size }
func (r *RemoteFileDataSourceModel) GetTimeout() types.String       { return r.Timeout }
func (r *RemoteFileDataSourceModel) GetTriggers() types.Map         { return r.Triggers }
func (r *RemoteFileDataSourceModel) GetUser() types.String          { return r.User }
func (r *RemoteFileDataSourceModel) GetID() types.String            { return r.ID }
func (r *RemoteFileDataSourceModel) GetRetryCount() types.Int64     { return r.RetryCount }
func (r *RemoteFileDataSourceModel) GetRetryInterval() types.String { return r.RetryInterval }
