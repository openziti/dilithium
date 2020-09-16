transwarp_protocol = Proto("Transwarp", "Transwarp Protocol")

seq = ProtoField.int32("transwarp.seq", "Sequence Number", base.DEC)
mt = ProtoField.uint8("transwarp.mt", "Message Type", base.HEX)

transwarp_protocol.fields = { seq, mt }

function transwarp_protocol.dissector(buffer, pinfo, tree)
	length = buffer:len()
	if length == 0 then return end

	pinfo.cols.protocol = transwarp_protocol.name

	local subtree = tree:add(transwarp_protocol, buffer(), "Transwarp Protocol Data")

	subtree:add(seq, buffer(0,4))
	local mt = buffer(4, 1):uint()
	local mt_name = get_mt_name(mt)
	subtree:add(mt, buffer(5,1)):append_text(" (" .. mt_name .. ")")
end

function get_mt_name(mt)
	local mt_name = "UNKNOWN"

	    if mt == 0 then mt_name = "HELLO"
	elseif mt == 1 then mt_name = "ACK"
	elseif mt == 2 then mt_name = "DATA"
	elseif mt == 3 then mt_name = "CLOSE" end

	return mt_name
end

local udp_port = DissectorTable.get("udp.port")
udp_port:add(6262, transwarp_protocol)