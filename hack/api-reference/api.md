<p>Packages:</p>
<ul>
<li>
<a href="#openstack.machine.gardener.cloud%2fv1alpha1">openstack.machine.gardener.cloud/v1alpha1</a>
</li>
</ul>
<h2 id="openstack.machine.gardener.cloud/v1alpha1">openstack.machine.gardener.cloud/v1alpha1</h2>
<p>
<p>Package v1alpha1 is a version of the API.</p>
</p>
Resource Types:
<ul></ul>
<h3 id="openstack.machine.gardener.cloud/v1alpha1.MachineProviderConfig">MachineProviderConfig
</h3>
<p>
<p>MachineProviderConfig contains OpenStack specific configuration for a machine.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>spec</code></br>
<em>
<a href="#openstack.machine.gardener.cloud/v1alpha1.MachineProviderConfigSpec">
MachineProviderConfigSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<br/>
<br/>
<table>
<tr>
<td>
<code>imageID</code></br>
<em>
string
</em>
</td>
<td>
<p>ImageID is the ID of image used by the machine.</p>
</td>
</tr>
<tr>
<td>
<code>imageName</code></br>
<em>
string
</em>
</td>
<td>
<p>ImageName is the name of the image used the machine. If ImageID is specified, it takes priority over ImageName.</p>
</td>
</tr>
<tr>
<td>
<code>region</code></br>
<em>
string
</em>
</td>
<td>
<p>Region is the region the machine should belong to.</p>
</td>
</tr>
<tr>
<td>
<code>availabilityZone</code></br>
<em>
string
</em>
</td>
<td>
<p>AvailabilityZone is the availability zone the machine belongs.</p>
</td>
</tr>
<tr>
<td>
<code>flavorName</code></br>
<em>
string
</em>
</td>
<td>
<p>FlavorName is the flavor of the machine.</p>
</td>
</tr>
<tr>
<td>
<code>keyName</code></br>
<em>
string
</em>
</td>
<td>
<p>KeyName is the name of the key pair used for SSH access.</p>
</td>
</tr>
<tr>
<td>
<code>securityGroups</code></br>
<em>
[]string
</em>
</td>
<td>
<p>SecurityGroups is a list of security groups the instance should belong to.</p>
</td>
</tr>
<tr>
<td>
<code>tags</code></br>
<em>
map[string]string
</em>
</td>
<td>
<p>Tags is a map of key-value pairs that annotate the instance. Tags are stored in the instance&rsquo;s Metadata field.</p>
</td>
</tr>
<tr>
<td>
<code>networkID</code></br>
<em>
string
</em>
</td>
<td>
<p>NetworkID is the ID of the network the instance should belong to.</p>
</td>
</tr>
<tr>
<td>
<code>subnetID</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>SubnetID is the ID of the subnet the instance should belong to. If SubnetID is not specified</p>
</td>
</tr>
<tr>
<td>
<code>subnetIDs</code></br>
<em>
[]string
</em>
</td>
<td>
<p>SubnetIDs is a list of IDs of the subnets the instance should belong to.</p>
</td>
</tr>
<tr>
<td>
<code>podNetworkCidr</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>PodNetworkCidr is the CIDR range for the pods assigned to this instance.
Deprecated: use PodNetworkCIDRs instead</p>
</td>
</tr>
<tr>
<td>
<code>podNetworkCIDRs</code></br>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>PodNetworkCIDRs is the CIDR ranges for the pods assigned to this instance.</p>
</td>
</tr>
<tr>
<td>
<code>rootDiskSize</code></br>
<em>
int
</em>
</td>
<td>
<p>The size of the root disk used for the instance.</p>
</td>
</tr>
<tr>
<td>
<code>rootDiskType</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The type of the root disk used for the instance.</p>
</td>
</tr>
<tr>
<td>
<code>useConfigDrive</code></br>
<em>
bool
</em>
</td>
<td>
<p>UseConfigDrive enables the use of configuration drives for the instance.</p>
</td>
</tr>
<tr>
<td>
<code>serverGroupID</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ServerGroupID is the ID of the server group this instance should belong to.</p>
</td>
</tr>
<tr>
<td>
<code>networks</code></br>
<em>
<a href="#openstack.machine.gardener.cloud/v1alpha1.OpenStackNetwork">
[]OpenStackNetwork
</a>
</em>
</td>
<td>
<p>Networks is a list of networks the instance should belong to. Networks is mutually exclusive with the NetworkID option
and only one should be specified.</p>
</td>
</tr>
</table>
</td>
</tr>
</tbody>
</table>
<h3 id="openstack.machine.gardener.cloud/v1alpha1.MachineProviderConfigSpec">MachineProviderConfigSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#openstack.machine.gardener.cloud/v1alpha1.MachineProviderConfig">MachineProviderConfig</a>)
</p>
<p>
<p>MachineProviderConfigSpec contains provider specific configuration for creating and managing machines.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>imageID</code></br>
<em>
string
</em>
</td>
<td>
<p>ImageID is the ID of image used by the machine.</p>
</td>
</tr>
<tr>
<td>
<code>imageName</code></br>
<em>
string
</em>
</td>
<td>
<p>ImageName is the name of the image used the machine. If ImageID is specified, it takes priority over ImageName.</p>
</td>
</tr>
<tr>
<td>
<code>region</code></br>
<em>
string
</em>
</td>
<td>
<p>Region is the region the machine should belong to.</p>
</td>
</tr>
<tr>
<td>
<code>availabilityZone</code></br>
<em>
string
</em>
</td>
<td>
<p>AvailabilityZone is the availability zone the machine belongs.</p>
</td>
</tr>
<tr>
<td>
<code>flavorName</code></br>
<em>
string
</em>
</td>
<td>
<p>FlavorName is the flavor of the machine.</p>
</td>
</tr>
<tr>
<td>
<code>keyName</code></br>
<em>
string
</em>
</td>
<td>
<p>KeyName is the name of the key pair used for SSH access.</p>
</td>
</tr>
<tr>
<td>
<code>securityGroups</code></br>
<em>
[]string
</em>
</td>
<td>
<p>SecurityGroups is a list of security groups the instance should belong to.</p>
</td>
</tr>
<tr>
<td>
<code>tags</code></br>
<em>
map[string]string
</em>
</td>
<td>
<p>Tags is a map of key-value pairs that annotate the instance. Tags are stored in the instance&rsquo;s Metadata field.</p>
</td>
</tr>
<tr>
<td>
<code>networkID</code></br>
<em>
string
</em>
</td>
<td>
<p>NetworkID is the ID of the network the instance should belong to.</p>
</td>
</tr>
<tr>
<td>
<code>subnetID</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>SubnetID is the ID of the subnet the instance should belong to. If SubnetID is not specified</p>
</td>
</tr>
<tr>
<td>
<code>subnetIDs</code></br>
<em>
[]string
</em>
</td>
<td>
<p>SubnetIDs is a list of IDs of the subnets the instance should belong to.</p>
</td>
</tr>
<tr>
<td>
<code>podNetworkCidr</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>PodNetworkCidr is the CIDR range for the pods assigned to this instance.
Deprecated: use PodNetworkCIDRs instead</p>
</td>
</tr>
<tr>
<td>
<code>podNetworkCIDRs</code></br>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>PodNetworkCIDRs is the CIDR ranges for the pods assigned to this instance.</p>
</td>
</tr>
<tr>
<td>
<code>rootDiskSize</code></br>
<em>
int
</em>
</td>
<td>
<p>The size of the root disk used for the instance.</p>
</td>
</tr>
<tr>
<td>
<code>rootDiskType</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The type of the root disk used for the instance.</p>
</td>
</tr>
<tr>
<td>
<code>useConfigDrive</code></br>
<em>
bool
</em>
</td>
<td>
<p>UseConfigDrive enables the use of configuration drives for the instance.</p>
</td>
</tr>
<tr>
<td>
<code>serverGroupID</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ServerGroupID is the ID of the server group this instance should belong to.</p>
</td>
</tr>
<tr>
<td>
<code>networks</code></br>
<em>
<a href="#openstack.machine.gardener.cloud/v1alpha1.OpenStackNetwork">
[]OpenStackNetwork
</a>
</em>
</td>
<td>
<p>Networks is a list of networks the instance should belong to. Networks is mutually exclusive with the NetworkID option
and only one should be specified.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="openstack.machine.gardener.cloud/v1alpha1.OpenStackNetwork">OpenStackNetwork
</h3>
<p>
(<em>Appears on:</em>
<a href="#openstack.machine.gardener.cloud/v1alpha1.MachineProviderConfigSpec">MachineProviderConfigSpec</a>)
</p>
<p>
<p>OpenStackNetwork describes a network this instance should belong to.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>id</code></br>
<em>
string
</em>
</td>
<td>
<p>Id is the ID of a network the instance should belong to.</p>
</td>
</tr>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>Name is the name of a network the instance should belong to. If Id is specified, it takes priority over Name.</p>
</td>
</tr>
<tr>
<td>
<code>podNetwork</code></br>
<em>
bool
</em>
</td>
<td>
<p>PodNetwork specifies whether this network is part of the pod network.</p>
</td>
</tr>
</tbody>
</table>
<hr/>
<p><em>
Generated with <a href="https://github.com/ahmetb/gen-crd-api-reference-docs">gen-crd-api-reference-docs</a>
</em></p>
