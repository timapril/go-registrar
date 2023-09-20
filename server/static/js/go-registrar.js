function isIPv4Address(addr){
  // Regular expression taken from
  // http://stackoverflow.com/questions/23483855/javascript-regex-to-validate-ipv4-and-ipv6-address-no-hostnames
  if (/^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$/.test(addr)) {
    return true;
  }
  return false;
}

function isIPv6Address(addr){
  // Regular expression taken from
  // http://stackoverflow.com/questions/23483855/javascript-regex-to-validate-ipv4-and-ipv6-address-no-hostnames
  if (/^\s*((([0-9A-Fa-f]{1,4}:){7}([0-9A-Fa-f]{1,4}|:))|(([0-9A-Fa-f]{1,4}:){6}(:[0-9A-Fa-f]{1,4}|((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3})|:))|(([0-9A-Fa-f]{1,4}:){5}(((:[0-9A-Fa-f]{1,4}){1,2})|:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3})|:))|(([0-9A-Fa-f]{1,4}:){4}(((:[0-9A-Fa-f]{1,4}){1,3})|((:[0-9A-Fa-f]{1,4})?:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){3}(((:[0-9A-Fa-f]{1,4}){1,4})|((:[0-9A-Fa-f]{1,4}){0,2}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){2}(((:[0-9A-Fa-f]{1,4}){1,5})|((:[0-9A-Fa-f]{1,4}){0,3}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){1}(((:[0-9A-Fa-f]{1,4}){1,6})|((:[0-9A-Fa-f]{1,4}){0,4}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(:(((:[0-9A-Fa-f]{1,4}){1,7})|((:[0-9A-Fa-f]{1,4}){0,5}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:)))(%.+)?\s*$/.test(addr)) {
    return true;
  }
  return false;
}

function getAddressType(addr) {
  // If the address is a v6 address, return 6
  if (isIPv6Address(addr)) { return "6"; }
  // If the address is a v4 address, return 4
  if (isIPv4Address(addr)) { return "4"; }
  // If we cant figure out what address type it is, return unknown
  return "unknown"
}

function isValidIPAddress(addr){
  // The address is valid if it is a v4 address or a v6 address
  return isIPv6Address(addr) || isIPv4Address(addr);
}

function host_entry_change_event(){
  var ip_addr = $("#to_add_address").val();
  var proto_title = $("#to_add_address_proto :selected").text();
  var selected_proto_type = ""

  // If nothing else, the message is empty but good
  var message = ""
  var goodMessage = true

  // Convert the selected protocol type to a 4 or a 6 so it will match
  // the addressType retunred from getAddressType
  if (proto_title == "v4") { selected_proto_type = "4"; }
  if (proto_title == "v6") { selected_proto_type = "6"; }

  // If the address is valid
  if (isValidIPAddress(ip_addr)) {

    // Get the up address type by checking the address itself
    ipType = getAddressType(ip_addr)

    // If the type was not unknonwn
    if (ipType != "unknown") {

      // Update the message to say that it was a valid IP address
      message = "&#x2714; - Valid IPv" + ipType + " address"

      // If the IP address type matches the select IP address type
      if ( selected_proto_type == ipType ) {

        // Add a message saying that the correct protocol is selected
        message = message + " / &#x2714; Correct Protocol Type"
      } else {

        // Otherwise, add a message saying that the incorrect protocol
        // is selected and set the goodMessage flag to false
        message = message + " / &#x2718; Incorrect Protocol Type"
        goodMessage = false
      }
    }
  } else {
    // If the address is not empty and also not a valid IPv4 or IPv6
    // address, display an invalid IP address message
    if (ip_addr.length != 0) {
      goodMessage = false
      message = "&#x2718; - Invalid IP Address"
    }
  }

  // Display the message
  $("#host_add_message").html(message)

  // Set the color of the message
  if ( goodMessage ) {
    $("#host_add_message").css("color", "#007A29")
  } else {
    $("#host_add_message").css("color", "#CC0000")
  }
}

function push_host_address() {
  var host_prefix = "host_address_";
  var id = $("#to_add_address_proto").val();
  var ip_addr = $("#to_add_address").val();
  var proto_title = $("#to_add_address_proto :selected").text();
  var edited = ip_addr.replace(/\./g, "G").replace(/:/g,"S");
  var host_id = host_prefix + proto_title + "-" + edited;
  var form_name = "host_address";

  // If the protocol type is valid (v4 or v6)
  if (id == "4" || id == "6") {
    // If the IP is not valid
    if (!isValidIPAddress(ip_addr)) {
      // log an error to the console
      console.log("Unable to parse IP address: " + ip_addr)
      return
    }

    // if the host does not exist in the list already, we should add it
    if ($("#" + host_id).length == 0) {

      // create the value that the server knows how to process
      storeValue = proto_title + "-" + ip_addr

      // Generate the link that can be used to remove the host from the
      // list
      var remove_link = "<a onclick='javascript:remove_host_addres(\"" + host_id + "\", \"" + ip_addr + "\", \"" + proto_title + "\");'>Remove</a>"

      // Generate the html line that will be added to the list
      var line_to_add = "<div id='" + host_id + "'><div class='form_name'></div><input type='hidden' id='"+ form_name + "' name='" + form_name + "' value='" + storeValue + "'><div class='title'>" + ip_addr + " - IP" + proto_title + "</div>&nbsp;" + remove_link +"</div>"

      // Add the line to the list of hosts list
      $("#host_address_list").html($("#host_address_list").html() + line_to_add)
    }else{
      // If the item is already in the list of hosts, see if it should
      // be readded (if it was removed)
      readd_host_address(host_id, ip_addr, proto_title)
    }

    // Clear the input box
    $("#to_add_address").val("")

    // Clear the display message box if there was any message
    host_entry_change_event()

  } else {
    // If the protocol was invalid, log a warning to the console
    console.log("Error: Unknown IP protocol number")
  }
}

function remove_host_addres(id, ip_addr, proto_title) {
  if ($("#" + id).length == 1) {
    // If the host address appears in the html list already
    var host_address_title = $("#" + id + " .title").text()

    // Create the link to undo the removal of the host address
    var readd_link = "<a onclick='javascript:readd_host_address(\"" + id + "\", \"" + ip_addr + "\", \"" + proto_title + "\");'>Undo</a>"

    // Create the HTML to swap into the existing body
    var line_to_add = "<div class='form_name'></div><div class='title'><strike>" + host_address_title + "</strike></div>&nbsp;" + readd_link + "</div>"

    // Swap the HTML (also removing the input tag)
    $("#" + id).html(line_to_add)
  } else {
    // The host address is not in the list currently, no need to remove
    console.log("Cannot remove host address with an ID of " + id + ". Host Address is not in the list of host addresses yet.")
  }
}

function readd_host_address(id, ip_addr, proto_title) {
  var form_name = "host_address";

  if ($("#" + id).length == 1) {
    storeValue = proto_title + "-" + ip_addr

    // If the host address appears in the html list already
    var host_address_title = $("#" + id + " .title").text()

    // Create the link to remove the host address
    var remove_link = "<a onclick='javascript:remove_host_addres(\"" + id + "\", \"" + ip_addr + "\", \"" + proto_title + "\");'>Remove</a>"

    // Create the HTML to swap into the existing body
    var line_to_add = "<div class='form_name'></div><input type='hidden' id='"+ form_name + "' name='" + form_name + "' value='" + storeValue + "'><div class='title'>" + ip_addr + " - IP" + proto_title + "</div>&nbsp;" + remove_link +"</div>"

    // Swap the HTML (also removing the input tag)
    $("#" + id).html(line_to_add)
  } else {
    // If the host address does not appear in the HTML already. It needs
    // to be added through the dropdown
    console.log("Error: please add host address for the first time through the dropdown")
  }
}

function push_ds_entry() {
  var ds_prefix = "ds_entry_";
  var form_name = "ds_entry";
  var key_tag = $("#to_add_key_tag").val();
  var algorithm = $("#to_add_alg :selected").val();
  var digest_type = $("#to_add_digest_type :selected").val();
  var digest = $("#to_add_digest").val().replace(/\s/g, '');

  if (IsValidKeyTag(key_tag) && IsValidDigest(digest)) {
    var ds_entry_string = key_tag + ":" + algorithm + ":" + digest_type + ":" + digest
    var ds_entry_id = ds_entry_string.replace(/:/g, "-")

    if ($("#" + ds_entry_id).length == 0) {
      var remove_link = "<a onclick='javascript:remove_ds_entry(\"" + ds_entry_id + "\", \"" + ds_entry_string + "\", \"" + key_tag + "\", \"" + algorithm + "\", \"" + digest_type + "\", \"" + digest + "\")'>Remove</a>";

      var line_to_add = "<div id='" + ds_entry_id + "'><div class='form_name'></div><input type='hidden' id='" + form_name + "' name='" + form_name + "' value='" + ds_entry_string + "'><div class='title'>" + ds_entry_string + "</div>&nbsp;" + remove_link + "</div>"

      $("#dnssec_list").html($("#dnssec_list").html() + line_to_add)
    }else{
      // If the item is already in the list of hosts, see if it should
      // be readded (if it was removed)
      readd_ds_entry(ds_entry_id, key_tag, algorithm, digest_type, digest)
    }

    // Clear the input box
    $("#to_add_key_tag").val("")
    $("#to_add_digest").val("")
  }
}

function remove_ds_entry(id, value, key_tag, algorithm, digest_type, digest) {
  if ($("#" + id).length == 1) {
    // If the ds entry appears in the html list already
    var ds_entry_title = $("#" + id + " .title").text()

    // Create the link to undo the removal of the ds entry
    var readd_link = "<a onclick='javascript:readd_ds_entry(\"" + id + "\", \"" + value + "\", \"" + key_tag + "\", \"" + algorithm + "\", \"" + digest_type + "\", \"" + digest + "\");'>Undo</a>"

    // Create the HTML to swap into the existing body
    var line_to_add = "<div class='form_name'></div><div class='title'><strike>" + ds_entry_title + "</strike></div>&nbsp;" + readd_link + "</div>"

    // Swap the HTML (also removing the input tag)
    $("#" + id).html(line_to_add)
  } else {
    // The ds entry is not in the list currently, no need to remove
    console.log("Cannot remove ds entry with an ID of " + id + ". DS Entry is not in the list of ds entries yet.")
  }
}

function readd_ds_entry(id, value, key_tag, algorithm, digest_type, digest) {
  var form_name = "ds_entry";

  if ($("#" + id).length == 1) {

    // If the ds entry appears in the html list already
    var ds_entry_title = $("#" + id + " .title").text()

    // Create the link to remove the ds entry
    var remove_link = "<a onclick='javascript:remove_ds_entry(\"" + id + "\",\"" + value + "\", \"" + key_tag + "\", \"" + algorithm + "\", \"" + digest_type + "\", \"" + digest + "\");'>Remove</a>"

    // Create the HTML to swap into the existing body
    var line_to_add = "<div class='form_name'></div><input type='hidden' id='"+ form_name + "' name='" + form_name + "' value='" + value + "'><div class='title'>" + value + "</div>&nbsp;" + remove_link +"</div>"

    // Swap the HTML (also removing the input tag)
    $("#" + id).html(line_to_add)
  } else {
    // If the host address does not appear in the HTML already. It needs
    // to be added through the dropdown
    console.log("Error: please add host address for the first time through the dropdown")
  }
}

function ds_entry_change_event(){
  var key_tag = $("#to_add_key_tag").val();
  var digest = $("#to_add_digest").val();

  // If nothing else, the message is empty but good
  var message = ""
  var goodMessage = true

  if (IsValidKeyTag(key_tag)) {
    message = "&#x2714; Valid Key Tag";
  } else {
    message = "&#x2718; Invalid Key Tag, must be between 0 and 65535";
    goodMessage = false;
  }

  if (IsValidDigest(digest)) {
    message = message + " / &#x2714; Valid Digest";
  } else {
    message = message + " / &#x2718; Invalid Digest";
    goodMessage = false;
  }

  // Display the message
  $("#ds_entry_add_message").html(message)

  // Set the color of the message
  if ( goodMessage ) {
    $("#ds_entry_add_message").css("color", "#007A29")
  } else {
    $("#ds_entry_add_message").css("color", "#CC0000")
  }
}

function IsValidKeyTag(kt) {
  var key_tag = Number(kt);
  if (key_tag % 1 === 0) {
    if (key_tag >= 0 && key_tag <= 65535) {
      return true
    }
  }
  return false
}

function IsValidDigest(digest) {
  if (/^[a-zA-Z0-9 ]*$/.test(digest) && digest.length > 0) {
    return true;
  }
  return false;
}

function push_hostname() {
  var id = $("#to_add_hostname").val()

  console.log(id)
  if (id != "none") {
    // First, check if the ID is actually an integer
    if (isNaN(parseInt(id))) {
      console.log("Unable to treat " + id + " as an INT. Stopping")
      return
    }

    if ($("#hostname_" + id).length == 0) {
      // If the hostname does not appear in the html list already,
      // grab the title from the dropdown
      var hostname_title = $('#to_add_hostname :selected').text();

      // Create the link to remove the hostname
      var remove_link = "<a onclick='javascript:remove_hostname(" + id + ");'>Remove</a>";

      // Create the HTML to push into the list
      var line_to_add = "<div id='hostname_" + id + "'><div class='form_name'></div><input type='hidden' id='hostname' name='hostname' value ='" + id + "'><div class='title'>" + hostname_title + "</div>&nbsp;" + remove_link + "</div>";

      // Add the HTML to the body of the page
      $("#hostname_list").html($("#hostname_list").html() + line_to_add)
    } else {
      // If the hostname already exists, lets readd it to the page
      // (there is no harm in readding it even if its already added)
      readd_hostname(id)
    }
  }else{
    console.log("Error: Cant add \"none\" as an approver set")
  }
}

function remove_hostname(id){
  // First, check if the ID is actually an integer
  if (isNaN(parseInt(id))) {
    console.log("Unable to treat " + id + " as an INT. Stopping.")
    return
  }

  if ($("#hostname_" + id).length > 0) {
    // If the hostname appears in the html list already, grab the title
    var hostname_title = $("#hostname_" + id + " .title").text()

    // Create the link to readd the hostname
    var readd_link = "<a onclick='javascript:readd_hostname(" + id + ");'>Undo</a>"

    // Create the HTML to swap into the existing body
    var line_to_add = "<div class='form_name'></div><div class='title'><strike>" + hostname_title + "</strike></div>&nbsp;" + readd_link + "</div>"

    // Swap the HTML (also removing the input tag)
    $("#hostname_" + id).html(line_to_add)
  } else {
    // The hostname is not in the list currently, no need to remove
    console.log("Cannot remove hostname with an ID of " + id + ". Hostname is not in the list of hostnames yet.")
  }
}

function readd_hostname(id) {
  // First, check if the ID is actually an integer
  if (isNaN(parseInt(id))) {
    console.log("Unable to treat " + id + " as an INT. Stopping.")
    return
  }

  if ($("#hostname_" + id).length > 0) {
    // If the hostname appears in the html list already, grab the title
    var hostname_title = $("#hostname_" + id + " .title").text()

    // Create the link to remove the hostname
    var remove_link = "<a onclick='javascript:remove_hostname(" + id + ");'>Remove</a>";

    // Create the HTML to swap into the existing body
    var line_to_add = "<div class='form_name'></div><input type='hidden' id='hostname' name='hostname' value ='" + id + "'><div class='title'>" + hostname_title + "</div>&nbsp;" + remove_link

    // Swap the HTML (also readding the input tag)
    $("#hostname_" + id).html(line_to_add)
  } else {
    // If the hostname does not appear in the HTML already. It needs to
    // be added through the dropdown
    console.log("Error: please add hostname for the first time through the dropdown")
  }
}

function push_approver_set(type) {
  // Verify the type of approver set
  if (type === "required" || type === "informed"){
    // Initialize the required variables
    var dropdown_id = ""
    var approver_set_prefix = ""

    if (type === "required") {
      dropdown_id = "#to_add_approver_required"
      approver_set_prefix = "approver_set_required_"
    }
    if (type === "informed") {
      dropdown_id = "#to_add_approver_informed"
      approver_set_prefix = "approver_set_informed_"
    }
    // Grab the current selected value from the dropdown
    var id = $(dropdown_id).val()

    if (id != "none") {
      // If the selected value is not "None" (or no selection)

      // First, check if the ID is actually an integer
      if (isNaN(parseInt(id))) {
        console.log("Unable to treat " + id + " as an INT. Stopping.")
        return
      }

      if ($("#" + approver_set_prefix + id).length == 0) {
        // If the approver set does not already exist in the HTML body

        // Grab the approver_set_title from the dropdown
        var approver_title = $(dropdown_id + ' :selected').text();

        // Create the link to remove the approver set
        var remove_link = "<a onclick='javascript:remove_approver_set(\"" + type + "\"," + id + ");'>Remove</a>"

        // Create the HTML to push into the list
        var line_to_add = "<div id='" + approver_set_prefix + id + "'><div class='form_name'></div><input type='hidden' id='"+ approver_set_prefix + "id' name='" + approver_set_prefix + "id' value='" + id + "'><div class='title'>" + approver_title + "</div>&nbsp;" + remove_link

        // Add the HTML to the body of the page
        $("#" + approver_set_prefix + "list ").html($("#" + approver_set_prefix + "list").html() + line_to_add)
      }else{
        // If the approver set already exists, lets readd it to the page
        // (there is no harm in readding it even if its already added)
        readd_approver_set(type, id)
      }
    }else{
      console.log("Error: Cant add \"None\" as an approver set")
    }
  } else {
    console.log("Approver set type unknown ( " + type + " )")
  }
}

function remove_approver_set(type, id) {
  // First, check if the ID is actually an integer
  if (isNaN(parseInt(id))) {
    console.log("Unable to treat " + id + " as an INT. Stopping.")
    return
  }

  // Verify the type of approver set
  if (type === "required" || type === "informed"){
    // Initialize the required variables
    var dropdown_id = ""
    var approver_set_prefix = ""

    if (type === "required") {
      dropdown_id = "#to_add_approver_required"
      approver_set_prefix = "approver_set_required_"
    }
    if (type === "informed") {
      dropdown_id = "#to_add_approver_informed"
      approver_set_prefix = "approver_set_informed_"
    }

    if ($("#" + approver_set_prefix + id).length > 0) {
      // If the approver set appears in the html list already
      var approver_title = $("#" + approver_set_prefix + id + " .title").text()

      // Create the link to undo the removal of the approver set
      var readd_link = "<a onclick='javascript:readd_approver_set(\"" + type + "\"," + id + ");'>Undo</a>"

      // Create the HTML to swap into the existing body
      var line_to_add = "<div class='form_name'></div><div class='title'><strike>" + approver_title + "</strike></div>&nbsp;" + readd_link + "</div>"

      // Swap the HTML (also removing the input tag)
      $("#" + approver_set_prefix + id).html(line_to_add)
    } else {
      // The approver set is not in the list currently, no need to remove
      console.log("Cannot remove approver set with an ID of " + id + ". Approver set is not in the list of approver sets yet.")
    }
  } else {
    console.log("Approver set type unknown ( " + type + " )")
  }
}

function readd_approver_set(type, id) {
  // First, check if the ID is actually an integer
  if (isNaN(parseInt(id))) {
    console.log("Unable to treat " + id + " as an INT. Stopping.")
    return
  }

  // Verify the type of approver set
  if (type === "required" || type === "informed"){
    // Initialize the required variables
    var dropdown_id = ""
    var approver_set_prefix = ""

    if (type === "required") {
      dropdown_id = "#to_add_approver_required"
      approver_set_prefix = "approver_set_required_"
    }
    if (type === "informed") {
      dropdown_id = "#to_add_approver_informed"
      approver_set_prefix = "approver_set_informed_"
    }

    if ($("#" + approver_set_prefix + id).length > 0) {
      // If the approver set appears in the html list already
      var approver_title = $("#" + approver_set_prefix + id + " .title").text()

      // Create the link to remove the approver set
      var remove_link = "<a onclick='javascript:remove_approver_set(\"" + type + "\"," + id + ");'>Remove</a>"

      // Create the HTML to swap into the existing body
      var line_to_add = "<div class='form_name'></div><input type='hidden' id='"+ approver_set_prefix + "id' name='" + approver_set_prefix + "id' value='" + id + "'><div class='title'>" + approver_title + "</div>&nbsp;" + remove_link

      // Swap the HTML (also readding the input tag)
      $("#" + approver_set_prefix + id).html(line_to_add)
    } else {
      // If the approver set does not appear in the HTML already. It needs to
      // be added through the dropdown
      console.log("Error: please add approver set for the first time through the dropdown")
    }
  } else {
    console.log("Approver set type unknown ( " + type + " )")
  }
}

function push_approver() {
  // Grab the current selected value from the dropdown
  var id = $('#to_add_approver').val()

  if (id != "none") {
    // If the selected value is not "None" (or no selection)

    // First, check if the ID is actually an integer
    if (isNaN(parseInt(id))) {
      console.log("Unable to treat " + id + " as an INT. Stopping.")
      return
    }

    if ($("#approver_" + id).length == 0) {
      // If the approver does not already exist in the HTML body

      // Grab the approver_title from the dropdown
      var approver_title = $('#to_add_approver :selected').text();

      // Create the link to remove the approver
      var remove_link = "<a onclick='javascript:remove_approver(" + id + ");'>Remove</a>"

      // Create the HTML to swap into the existing body
      var line_to_add = "<div id='approver_"+ id + "'><div class='form_name'></div><input type='hidden' id='approver_id' name='approver_id' value='" + id + "'><div class='emailRole'>" + approver_title + "</div>&nbsp;" + remove_link +"</div></br>"

      // Add the HTML to the body of the page
      $("#approverlist").html($("#approverlist").html() + line_to_add)
    }else{
      // If the approver already exists, lets readd it to the page
      // (there is no harm in readding it even if its already added)
      readd_approver(id)
    }
  }else{
    console.log("Error: Cant add \"None\" as an approver")
  }
  // TODO: Set the selected dropdown to 0
}

function remove_approver(id) {
  // First, check if the ID is actually an integer
  if (isNaN(parseInt(id))) {
    console.log("Unable to treat " + id + " as an INT. Stopping.")
    return
  }

  if ($("#approver_" + id).length > 0) {
    // If the approver appears in the html list already
    var approver_title = $("#approver_" + id + " .emailRole").text()

    // Create the link to undo the removal of the approver
    var readd_link = "<a onclick='javascript:readd_approver(" + id + ");'>Undo</a>"

    // Create the HTML to swap into the existing body
    var line_to_add = "<div class='form_name'></div><div class='emailRole'><strike>" + approver_title + "</strike></div>&nbsp;" + readd_link

    // Swap the HTML (also removing the input tag)
    $("#approver_" + id).html(line_to_add)
  } else {
    // The approver is not in the list currently, no need to remove
    console.log("Cannot remove approver with an ID of " + id + ". Approver is not in the list of approvers yet.")
  }
}

function readd_approver(id) {
  // First, check if the ID is actually an integer
  if (isNaN(parseInt(id))) {
    console.log("Unable to treat " + id + " as an INT. Stopping.")
    return
  }

  if ($("#approver_" + id).length > 0) {
    // If the approver does exist in the HTML, we can just replace it
    var approver_title = $("#approver_" + id + " .emailRole").text()

    // Create the link to remove the approver
    var remove_link = "<a onclick='javascript:remove_approver(" + id + ");'>Remove</a>"

    // Create the HTML to swap into the existing body
    var line_to_add = "<div class='form_name'></div><input type='hidden' id='approver_id' name='approver_id' value='" + id + "'><div class='emailRole'>" + approver_title + "</div>&nbsp;" + remove_link

    // Swap the HTML (also removing the input tag)
    $("#approver_" + id).html(line_to_add)
  } else {
    // If the approver does not appear in the HTML already. It needs to
    // be added through the dropdown
    console.log("Error: please add approvers for the first time through the dropdown")

  }
}

function toggle_content(id_prefix) {
  // Get the object handles for the two html elements that will be
  // change
  var action_div = $("#" + id_prefix + "Action")
  var content_div = $("#" + id_prefix + "Content")

  // Change the message of the expand/collapse tag to indicate that the
  // action is going to take place
  if (content_div.is(':hidden')){
    action_div.html("Expanding");
  } else {
    action_div.html("Collapsing");
  }

  // Make the chang to the object (expand/collapse)
  content_div.slideToggle("slow", function(){
      // When the change is done, change the message to the other option
      if (content_div.is(':hidden')){
        action_div.html("Expand");
      } else {
        action_div.html("Collapse");
      }
    });
}

// update_domain_class is used to update the other field for the domain
// class selection when other is either selected or not
function update_domain_class() {
  var otherField = $("#domain_class_other");
  if ($('#domain_class :selected').val() === "other") {
    if (otherField.is(":hidden")) {
      otherField.slideToggle("fast")
    }
  } else {
    otherField[0].value="";
    console.log(otherField.value)
    if (!otherField.is(":hidden")) {
      otherField.slideToggle("fast")
    }
  }
}
