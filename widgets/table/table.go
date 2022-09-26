package table

import (
	"fmt"
	"path/filepath"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/gou/lang"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
	"github.com/yaoapp/yao/widgets/component"
)

//
// API:
//   GET  /api/__yao/table/:id/setting  					-> Default process: yao.table.Xgen
//   GET  /api/__yao/table/:id/search  						-> Default process: yao.table.Search $param.id :query $query.page  $query.pagesize
//   GET  /api/__yao/table/:id/get  						-> Default process: yao.table.Get $param.id :query
//   GET  /api/__yao/table/:id/find/:primary  				-> Default process: yao.table.Find $param.id $param.primary :query
//   GET  /api/__yao/table/:id/component/:xpath/:method  	-> Default process: yao.table.Component $param.id $param.xpath $param.method :query
//  POST  /api/__yao/table/:id/save  						-> Default process: yao.table.Save $param.id :payload
//  POST  /api/__yao/table/:id/create  						-> Default process: yao.table.Create $param.id :payload
//  POST  /api/__yao/table/:id/insert  						-> Default process: yao.table.Insert :payload
//  POST  /api/__yao/table/:id/update/:primary  			-> Default process: yao.table.Update $param.id $param.primary :payload
//  POST  /api/__yao/table/:id/update/where  				-> Default process: yao.table.UpdateWhere $param.id :query :payload
//  POST  /api/__yao/table/:id/update/in  					-> Default process: yao.table.UpdateIn $param.id $query.ids :payload
//  POST  /api/__yao/table/:id/delete/:primary  			-> Default process: yao.table.Delete $param.id $param.primary
//  POST  /api/__yao/table/:id/delete/where  				-> Default process: yao.table.DeleteWhere $param.id :query
//  POST  /api/__yao/table/:id/delete/in  					-> Default process: yao.table.DeleteIn $param.id $query.ids
//
// Process:
// 	 yao.table.Setting Return the App DSL
// 	 yao.table.Xgen Return the Xgen setting
//   yao.table.Search Return the records with pagination
//   yao.table.Get  Return the records without pagination
//   yao.table.Find Return the record via the given primary key
//   yao.table.Component Return the result defined in props.xProps
//   yao.table.Save Save a record, if given a primary key update, else insert
//   yao.table.Create Create a record
//   yao.table.Insert Insert records
//   yao.table.Update update record via the given primary key
//   yao.table.UpdateWhere update record via the given query params
//   yao.table.UpdateIn update record via the given primary key list
//   yao.table.Delete delete record via the given primary key
//   yao.table.DeleteWhere delete record via the given query params
//   yao.table.DeleteIn delete record via the given primary key list
//
// Hook:
//   before:find
//   after:find
//   before:search
//   after:search
//   before:get
//   after:get
//   before:save
//   after:save
//   before:create
//   after:create
//   before:delete
//   after:delete
//   before:insert
//   after:insert
//   before:delete-in
//   after:delete-in
//   before:delete-where
//   after:delete-where
//   before:update-in
//   after:update-in
//   before:update-where
//   after:update-where
//

// Tables the loaded table widgets
var Tables map[string]*DSL = map[string]*DSL{}

// New create a new DSL
func New(id string) *DSL {
	return &DSL{
		ID:          id,
		CProps:      map[string]component.CloudPropsDSL{},
		ComputesIn:  map[string]string{},
		ComputesOut: map[string]string{},
	}
}

// Load load task
func Load(cfg config.Config) error {
	var root = filepath.Join(cfg.Root, "tables")
	return LoadFrom(root, "")
}

// LoadFrom load from dir
func LoadFrom(dir string, prefix string) error {

	if share.DirNotExists(dir) {
		return fmt.Errorf("%s does not exists", dir)
	}

	messages := []string{}
	err := share.Walk(dir, ".json", func(root, filename string) {
		id := prefix + share.ID(root, filename)
		data := share.ReadFile(filename)
		dsl := New(id)
		err := jsoniter.Unmarshal(data, dsl)
		if err != nil {
			messages = append(messages, fmt.Sprintf("[%s] %s", id, err.Error()))
			return
		}

		if dsl.Action == nil {
			dsl.Action = &ActionDSL{}
		}
		dsl.Action.SetDefaultProcess()

		if dsl.Layout == nil {
			dsl.Layout = &LayoutDSL{}
		}

		if dsl.Fields == nil {
			dsl.Fields = &FieldsDSL{}
		}

		// Bind model / store / table / ...
		err = dsl.Bind()
		if err != nil {
			messages = append(messages, fmt.Sprintf("[%s] %s", id, err.Error()))
			return
		}

		// Parse
		err = dsl.Parse()
		if err != nil {
			messages = append(messages, fmt.Sprintf("[%s] %s", id, err.Error()))
			return
		}

		// Validate
		err = dsl.Validate()
		if err != nil {
			messages = append(messages, fmt.Sprintf("[%s] %s", id, err.Error()))
			return
		}

		// Apply a language pack
		if lang.Default != nil {
			lang.Default.Apply(dsl)
		}

		Tables[id] = dsl
	})

	if len(messages) > 0 {
		return fmt.Errorf(strings.Join(messages, ";"))
	}

	return err
}

// Get table via process or id
func Get(table interface{}) (*DSL, error) {
	id := ""
	switch table.(type) {
	case string:
		id = table.(string)
	case *gou.Process:
		id = table.(*gou.Process).ArgsString(0)
	default:
		return nil, fmt.Errorf("%v type does not support", table)
	}

	t, has := Tables[id]
	if !has {
		return nil, fmt.Errorf("%s does not exist", id)
	}
	return t, nil
}

// MustGet Get table via process or id thow error
func MustGet(table interface{}) *DSL {
	t, err := Get(table)
	if err != nil {
		exception.New(err.Error(), 400).Throw()
	}
	return t
}

// Parse Layout
func (dsl *DSL) Parse() error {

	// init
	if dsl.Fields == nil {
		dsl.Fields = &FieldsDSL{
			Filter: map[string]FilterFiledsDSL{},
			Table:  map[string]ViewFiledsDSL{},
		}
	}
	return dsl.parseProps()
}

// Xgen trans to xgen setting
func (dsl *DSL) Xgen() (map[string]interface{}, error) {

	setting, err := dsl.Layout.Xgen()
	if err != nil {
		return nil, err
	}

	fields, err := dsl.Fields.Xgen()
	if err != nil {
		return nil, err
	}

	setting["fields"] = fields
	for _, cProp := range dsl.CProps {
		err := cProp.Replace(setting, func(cProp component.CloudPropsDSL) interface{} {
			return map[string]interface{}{
				"api":    fmt.Sprintf("/api/__yao/table/%s/component/%s/%s", dsl.ID, cProp.Xpath, cProp.Name),
				"params": cProp.Query,
			}
		})

		if err != nil {
			return nil, err
		}
	}

	return setting, nil
}

// parseCloudProps parse the props
func (dsl *DSL) parseProps() error {

	// filter
	for name, filter := range dsl.Fields.Filter {
		if filter.Edit != nil && filter.Edit.Props != nil {
			xpath := fmt.Sprintf("fields.filter.%s.edit.props", name)
			cProps, err := filter.Edit.Props.CloudProps(xpath)
			if err != nil {
				return err
			}
			dsl.mergeCProps(cProps)
		}
	}

	// table
	for name, column := range dsl.Fields.Table {

		// Computes
		if column.In != "" {
			dsl.ComputesIn[column.Bind] = column.In
			dsl.ComputesIn[name] = column.In
		}

		if column.Out != "" {
			dsl.ComputesOut[column.Bind] = column.Out
			dsl.ComputesOut[name] = column.Out
		}

		// Cloud Props
		if column.View != nil && column.View.Props != nil {
			xpath := fmt.Sprintf("fields.table.%s.view.props", name)
			cProps, err := column.View.Props.CloudProps(xpath)
			if err != nil {
				return err
			}
			dsl.mergeCProps(cProps)
		}

		if column.Edit != nil && column.Edit.Props != nil {
			xpath := fmt.Sprintf("fields.table.%s.edit.props", name)
			cProps, err := column.Edit.Props.CloudProps(xpath)
			if err != nil {
				return err
			}
			dsl.mergeCProps(cProps)
		}
	}

	return nil
}

func (dsl *DSL) mergeCProps(cProps map[string]component.CloudPropsDSL) {
	for k, v := range cProps {
		dsl.CProps[k] = v
	}
}
