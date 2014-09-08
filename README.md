# OneDrive Client

This is a basic client for uploading and downloading files to/from Microsoft OneDrive.
You will need to perform OAuth authentication yourself - see [MSDN documentation](http://msdn.microsoft.com/en-us/library/dn631818.aspx).

Files and folders in OneDrive are referenced by node id. If you want to reference them by path you will have to use the `ResolvePath` method. Then you can stat the node (`NodeInfo`) or list its children (`NodeFiles`).

Methods `Upload` and `Download` perform streming uploads and downloads to desired nodes.