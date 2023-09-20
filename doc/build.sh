#!/bin/bash

dot -Tpng approval_states.dot -o approval_states.png
dot -Tpng approver_states.dot -o approver_states.png
dot -Tpng approverrevision_states.dot -o approverrevision_states.png
dot -Tpng approverset_states.dot -o approverset_states.png
dot -Tpng approversetrevision_states.dot -o approversetrevision_states.png
dot -Tpng changerequest_states.dot -o changerequest_states.png
dot -Tpng domain_states.dot -o domain_states.png
