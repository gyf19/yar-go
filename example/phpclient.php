<?php
$client = new Yar_Client('tcp://127.0.0.1:12345');
$arguments =  ['A'=>4,'B'=>5,'C'=>'php'];
$data = $client->__call("Arith.Multiply",$arguments);
var_dump($data);
?>