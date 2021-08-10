westworld2_protocol = Proto("Westworld2", "Westworld2 Protocol")

seq = ProtoField.int32("westworld2.seq", "Sequence Number", base.DEC)
mt = ProtoField.uint8("westworld2.mt", "Message Type", base.HEX)
mf = ProtoField.uint8("westworld2.mf", "Message Flag", base.BIN)
ack = ProtoField.int32("westworld2.ack", "ACK", base.DEC)
westworld2_protocol.fields = { seq, mt, mf, ack }

function westworld2_protocol.dissector(buffer, pinfo, tree)
	length = buffer:len()
	if length == 0 then return end

	pinfo.cols.protocol = westworld2_protocol.name

	local subtree = tree:add(westworld2_protocol, buffer(), "Westworld2 Protocol")

	subtree:add(seq, buffer(0,4))
	local mt_v = buffer(4, 1):uint()
	local mt_name = get_mt_name(mt_v)
	subtree:add(mt, buffer(4,1)):append_text(" (" .. mt_name .. ")")
	subtree:add(mf, buffer(5,1))
	subtree:add(ack, buffer(6,4))
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
udp_port:add(6262, westworld2_protocol)