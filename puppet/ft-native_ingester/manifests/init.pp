class native_ingester {

  $binary_name = "native-ingester"
  $install_dir = "/usr/local/$binary_name"
  $binary_file = "$install_dir/$binary_name"
  $log_dir = "/var/log/apps"

  $src_addr = hiera('src_addr')
  $src_group = hiera('src_group')
  $src_topic = hiera('src_topic')
  $src_queue = hiera('src_queue')
  $src_concurrent = hiera('src_concurrent')
  $src_uuid_field = hiera('src_uuid_fields')
  $dest_address = hiera('dest_address')
  $dest_collections_by_origins = hiera('dest_collections_by_origins')
  $dest_header = hiera('dest_header')

  class { 'common_pp_up': }
  class { "${module_name}::monitoring": }
  class { "${module_name}::supervisord": }

  Class["${module_name}::supervisord"] -> Class['common_pp_up'] -> Class["${module_name}::monitoring"]

  user { $binary_name:
    ensure    => present,
  }

  file {
    $install_dir:
      mode    => "0664",
      ensure  => directory;

    $binary_file:
      ensure  => present,
      source  => "puppet:///modules/$module_name/$binary_name",
      mode    => "0755",
      require => File[$install_dir];

    $log_dir:
      ensure  => directory,
      mode    => "0664"
  }

  exec { 'restart_app':
    command     => "supervisorctl restart $binary_name",
    path        => "/usr/bin:/usr/sbin:/bin",
    subscribe   => [
      File[$binary_file],
      Class["${module_name}::supervisord"]
    ],
    refreshonly => true
  }
}
