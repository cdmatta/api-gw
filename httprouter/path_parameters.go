package httprouter

type PathParameter struct {
	Key   string
	Value string
}

type PathParameters struct {
	route      string
	parameters []PathParameter
}

func NewPathParameters(route string, parameterCount uint8) *PathParameters {
	return &PathParameters{
		route:      route,
		parameters: make([]PathParameter, 0, parameterCount),
	}
}

func (p *PathParameters) GetRoute() string {
	return p.route
}

func (p *PathParameters) GetParameters() []PathParameter {
	return p.parameters
}

func (p *PathParameters) AddParameter(key, value string) *PathParameters {
	parameterCount := len(p.parameters)
	p.parameters = p.parameters[:parameterCount+1]

	parameterIndex := parameterCount
	p.parameters[parameterIndex].Key = key
	p.parameters[parameterIndex].Value = value

	return p
}

func (p *PathParameters) ParameterMap() map[string]string {
	parameterMap := make(map[string]string)
	for index := range p.parameters {
		parameter := p.parameters[index]
		parameterMap[parameter.Key] = parameter.Value
	}
	return parameterMap
}
