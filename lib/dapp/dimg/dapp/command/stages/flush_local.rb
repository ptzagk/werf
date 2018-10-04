module Dapp
  module Dimg
    module Dapp
      module Command
        module Stages
          module FlushLocal
            def stages_flush_local
              ruby2go_cleanup_command(:flush, ruby2go_cleanup_stages_flush_local_options)
            end

            def ruby2go_cleanup_stages_flush_local_options
              {
                common_project_options: {
                  project_name: name,
                  common_options: {
                    dry_run: dry_run?,
                    force: true
                  }
                },
                with_dimgs: false,
                with_stages: true,
                only_repo: false,
              }.tap do |json|
                break JSON.dump(json)
              end
            end
          end
        end
      end
    end
  end # Dimg
end # Dapp
